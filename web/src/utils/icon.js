/**
 * 图标渲染工具
 *
 * 支持三种图标格式：
 * 1. 图片 URL   — 以 http:// 或 https:// 开头 → 渲染为 <img>
 * 2. CSS 图标类  — el-icon-xxx / fa fa-xxx / material-icons 等 → 渲染为 <i>
 * 3. 纯文本/emoji — 直接显示
 */

/** 判断是否为图片 URL */
export function isImageIcon(icon) {
  if (!icon) return false
  return /^https?:\/\//.test(icon) || /\.(png|jpg|jpeg|gif|svg|ico|webp)(\?.*)?$/i.test(icon)
}

/** 根据图标类型返回合适的背景渐变色 */
export function iconGradient(icon) {
  if (!icon) return 'linear-gradient(135deg, #409eff, #337ecc)'
  const map = {
    'el-icon-setting':    'linear-gradient(135deg, #667eea, #764ba2)',
    'el-icon-user':       'linear-gradient(135deg, #409eff, #337ecc)',
    'el-icon-s-custom':   'linear-gradient(135deg, #f093fb, #f5576c)',
    'el-icon-lock':       'linear-gradient(135deg, #4facfe, #00f2fe)',
    'el-icon-s-grid':     'linear-gradient(135deg, #43e97b, #38f9d7)',
    'el-icon-s-data':     'linear-gradient(135deg, #fa709a, #fee140)',
    'el-icon-s-order':    'linear-gradient(135deg, #a18cd1, #fbc2eb)',
    'el-icon-s-tools':    'linear-gradient(135deg, #fccb90, #d57eeb)',
    'el-icon-s-shop':     'linear-gradient(135deg, #ffecd2, #fcb69f)',
    'el-icon-s-finance':  'linear-gradient(135deg, #a1c4fd, #c2e9fb)',
    'el-icon-s-marketing': 'linear-gradient(135deg, #ff9a9e, #fecfef)',
  }
  if (map[icon]) return map[icon]
  // 图片 URL 用浅灰背景
  if (isImageIcon(icon)) return 'transparent'
  // 其他 CSS 图标类
  return 'linear-gradient(135deg, #409eff, #337ecc)'
}

/** 获取图标的 fallback 类名 */
export function fallbackIcon(icon) {
  return icon || 'el-icon-menu'
}
