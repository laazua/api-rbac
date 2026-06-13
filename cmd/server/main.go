package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "api-rbac/docs"

	"api-rbac/config"
	"api-rbac/internal/cache"
	"api-rbac/internal/handler"
	"api-rbac/internal/model"
	"api-rbac/internal/repository"
	"api-rbac/internal/router"
	"api-rbac/internal/service"
	jwtpkg "api-rbac/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// @title           RBAC 权限管理系统 API
// @version         1.0
// @description     通用的 RBAC 权限管理微服务，支持用户/角色/权限的 CRUD 及绑定关系管理。
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer {token} 格式的 JWT Token

func main() {
	// 解析命令行参数
	configPath := flag.String("c", "config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置 Gin 运行模式
	gin.SetMode(cfg.Server.Mode)

	// 初始化 JWT
	jwtpkg.Init(cfg.JWT.Secret, cfg.JWT.ExpireHour, cfg.JWT.RefreshExpireDay)

	// 连接数据库
	dbLogger := logger.Default.LogMode(logger.Info)
	if cfg.Server.Mode == gin.ReleaseMode {
		dbLogger = logger.Default.LogMode(logger.Warn)
	}

	db, err := gorm.Open(mysql.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 初始化 Redis（可选，连接失败仅日志警告不退出）
	var permCache *cache.PermissionCache
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("⚠️  Redis 连接失败 (%v)，权限缓存已禁用，将直接查询数据库", err)
	} else {
		permCache = cache.NewPermissionCache(rdb, 5*time.Minute)
		log.Println("✅ Redis 连接成功，权限缓存已启用")
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.ServiceAccount{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 初始化超级管理员 (首次运行)
	if err := initSuperAdmin(db); err != nil {
		log.Fatalf("超级管理员初始化失败: %v", err)
	}

	// 初始化 Repository
	userRepo := repository.NewUserRepo(db)
	roleRepo := repository.NewRoleRepo(db)
	permRepo := repository.NewPermissionRepo(db)
	saRepo := repository.NewServiceAccountRepo(db)

	// 初始化 Service
	authService := service.NewAuthService(userRepo)
	permCheckService := service.NewPermissionCheckService(userRepo, permCache)
	userService := service.NewUserService(userRepo, roleRepo, permCache)
	roleService := service.NewRoleService(roleRepo, permRepo, permCache)
	permService := service.NewPermissionService(permRepo)
	saService := service.NewServiceAccountService(saRepo)

	// 初始化 Handler
	authH := handler.NewAuthHandler(authService, permCheckService)
	userH := handler.NewUserHandler(userService)
	roleH := handler.NewRoleHandler(roleService)
	permH := handler.NewPermissionHandler(permService)
	saH := handler.NewServiceAccountHandler(saService)

	// 初始化默认 Service Account (首次运行)
	if err := initDefaultServiceAccount(saService); err != nil {
		log.Printf("⚠️  默认服务账号初始化失败: %v", err)
	}

	// 设置路由
	r := router.Setup(authH, userH, roleH, permH, saH, cfg, permCheckService, saRepo)

	// 创建 HTTP Server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 启动服务 (goroutine，非阻塞)
	go func() {
		log.Printf("服务启动于 http://0.0.0.0%s", addr)
		log.Printf("Swagger 文档: http://localhost%s/swagger/index.html", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 优雅关闭
	gracefulShutdown(srv)
}

// gracefulShutdown 等待退出信号，优雅关闭服务
func gracefulShutdown(srv *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("收到信号 %v，正在优雅关闭...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("服务强制关闭: %v", err)
	}

	log.Println("服务已关闭")
}

// initSuperAdmin 首次运行时初始化超级管理员
func initSuperAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		return fmt.Errorf("查询admin用户失败: %w", err)
	}

	if count > 0 {
		log.Println("[初始化] 超级管理员已存在，跳过")
		return nil
	}

	log.Println("========================================")
	log.Println("  检测到首次运行，请设置超级管理员密码")
	log.Println("========================================")

	password, err := readPasswordFromTerminal()
	if err != nil {
		return fmt.Errorf("读取密码失败: %w", err)
	}

	// 在事务中完成所有初始化，保证原子性
	err = db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建通配符权限
		perm := model.Permission{
			Name:        "超级管理员权限",
			Resource:    "*",
			Action:      "*",
			Description: "拥有所有资源的所有操作权限",
		}
		if err := tx.Create(&perm).Error; err != nil {
			return fmt.Errorf("创建权限失败: %w", err)
		}

		// 2. 创建超级管理员角色
		role := model.Role{
			Name:        "超级管理员",
			Description: "内置超级管理员角色，拥有全部权限",
		}
		if err := tx.Create(&role).Error; err != nil {
			return fmt.Errorf("创建角色失败: %w", err)
		}

		// 3. 绑定权限到角色
		if err := tx.Model(&role).Association("Permissions").Append(&perm); err != nil {
			return fmt.Errorf("绑定权限到角色失败: %w", err)
		}

		// 4. 创建 admin 用户
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("密码加密失败: %w", err)
		}

		user := model.User{
			Username: "admin",
			Password: string(hashedPassword),
			Email:    "admin@localhost",
			Status:   1,
		}
		if err := tx.Create(&user).Error; err != nil {
			return fmt.Errorf("创建admin用户失败: %w", err)
		}

		// 5. 绑定角色到用户
		if err := tx.Model(&user).Association("Roles").Append(&role); err != nil {
			return fmt.Errorf("绑定角色到用户失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Println("========================================")
	log.Println("  ✅ 超级管理员初始化完成")
	log.Println("     用户名: admin")
	log.Println("========================================")
	return nil
}

// readPasswordFromTerminal 从终端交互式读取密码，不回显
func readPasswordFromTerminal() (string, error) {
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		fmt.Print("请输入超级管理员密码 (不少于6位): ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("读取密码失败: %w", err)
		}
		fmt.Println()

		password := strings.TrimSpace(string(passwordBytes))
		if len(password) < 6 {
			fmt.Println("❌ 密码长度不能少于6位，请重新输入")
			continue
		}

		fmt.Print("请再次输入确认密码: ")
		confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("读取确认密码失败: %w", err)
		}
		fmt.Println()

		confirm := strings.TrimSpace(string(confirmBytes))

		if password != confirm {
			fmt.Printf("❌ 两次输入不一致，请重新输入 (剩余 %d 次)\n", maxRetries-i-1)
			continue
		}

		return password, nil
	}

	return "", fmt.Errorf("超过最大重试次数")
}

// initDefaultServiceAccount 首次运行时创建一个默认服务账号
func initDefaultServiceAccount(saService *service.ServiceAccountService) error {
	// 尝试列表查询来检测是否已有服务账号
	_, total, err := saService.List(&model.ListServiceAccountRequest{Page: 1, PageSize: 1})
	if err != nil {
		return fmt.Errorf("查询服务账号失败: %w", err)
	}

	if total > 0 {
		log.Println("[初始化] 服务账号已存在，跳过")
		return nil
	}

	sa, apiKey, err := saService.Create(&model.CreateServiceAccountRequest{
		Name:        "default",
		Description: "默认服务账号，用于服务间 API 调用",
	})
	if err != nil {
		return fmt.Errorf("创建默认服务账号失败: %w", err)
	}

	log.Println("========================================")
	log.Println("  ✅ 默认服务账号已创建")
	log.Printf("     Name: %s", sa.Name)
	log.Printf("     API Key: %s", apiKey)
	log.Println("     请妥善保管 API Key，此信息仅显示一次")
	log.Println("========================================")
	return nil
}
