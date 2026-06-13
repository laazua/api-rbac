/**
 * RBAC 权限管理系统 - Node.js SDK
 *
 * 用于从 Node.js 业务系统集成 RBAC 微服务，提供用户认证、权限检查等功能。
 *
 * @example
 * const { RBACClient } = require('./src');
 *
 * const client = new RBACClient('http://localhost:8087/api/v1');
 *
 * // 登录
 * const result = await client.login('admin', 'password');
 * const token = result.token;
 *
 * // 验证 Token
 * const user = await client.verify(token);
 *
 * // 检查权限
 * const allowed = await client.checkPermission(token, 'user', 'delete');
 *
 * // 批量检查
 * const results = await client.batchCheck(token, [
 *   ['user', 'read'],
 *   ['user', 'delete'],
 * ]);
 *
 * // Token 自省
 * const info = await client.introspect(token, 'order', 'read');
 */

class RBACClient {
  /**
   * @param {string} baseUrl - RBAC 服务地址，如 "http://localhost:8087/api/v1"
   * @param {number} [timeout=10000] - 请求超时毫秒数
   */
  constructor(baseUrl, timeout = 10000) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.timeout = timeout;
  }

  /**
   * 发送 POST 请求
   * @private
   */
  async _post(path, data = null, headers = {}) {
    const url = `${this.baseUrl}${path}`;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeout);

    try {
      const options = {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...headers },
        signal: controller.signal,
      };
      if (data) {
        options.body = JSON.stringify(data);
      }

      const response = await fetch(url, options);
      const result = await response.json();
      return result;
    } catch (err) {
      if (err.name === 'AbortError') {
        throw new Error(`请求超时: ${url}`);
      }
      throw new Error(`请求失败: ${err.message}`);
    } finally {
      clearTimeout(timer);
    }
  }

  /**
   * 用户登录
   * @param {string} account
   * @param {string} password
   * @returns {Promise<{token: string, refresh_token: string, expires_in: number, user_id: number, username: string}>}
   */
  async login(account, password) {
    const resp = await this._post('/auth/login', { account, password });
    if (resp.code !== 0) throw new Error(resp.message || '登录失败');
    return resp.data;
  }

  /**
   * 刷新 Token
   * @param {string} refreshToken
   * @returns {Promise<{token: string, refresh_token: string, expires_in: number}>}
   */
  async refresh(refreshToken) {
    const resp = await this._post('/auth/refresh', { refresh_token: refreshToken });
    if (resp.code !== 0) throw new Error(resp.message || '刷新失败');
    return resp.data;
  }

  /**
   * 验证 Token 有效性
   * @param {string} token
   * @returns {Promise<{user_id: number, username: string}>}
   */
  async verify(token) {
    const headers = { Authorization: `Bearer ${token}` };
    const resp = await this._post('/auth/verify', null, headers);
    if (resp.code !== 0) throw new Error(resp.message || 'Token 无效');
    return resp.data;
  }

  /**
   * 检查用户是否有指定权限
   * @param {string} token
   * @param {string} resource
   * @param {string} action
   * @returns {Promise<boolean>}
   */
  async checkPermission(token, resource, action) {
    const headers = { Authorization: `Bearer ${token}` };
    const resp = await this._post('/auth/check', { resource, action }, headers);
    if (resp.code === 1003) return false; // Forbidden
    if (resp.code !== 0) throw new Error(resp.message || '权限检查失败');
    return resp.data?.allowed ?? false;
  }

  /**
   * 批量检查权限
   * @param {string} token
   * @param {Array<[string, string]>} permissions - 如 [["user", "read"], ["user", "delete"]]
   * @returns {Promise<Object<string, boolean>>} - 如 {"user:read": true, "user:delete": false}
   */
  async batchCheck(token, permissions) {
    const headers = { Authorization: `Bearer ${token}` };
    const items = permissions.map(([resource, action]) => ({ resource, action }));
    const resp = await this._post('/auth/batch-check', { permissions: items }, headers);
    if (resp.code !== 0) throw new Error(resp.message || '批量权限检查失败');
    return resp.data?.results ?? {};
  }

  /**
   * Token 自省：验证 Token + 可选权限检查
   * @param {string} token
   * @param {string} [resource='']
   * @param {string} [action='']
   * @returns {Promise<{active: boolean, user_id?: number, username?: string}>}
   */
  async introspect(token, resource = '', action = '') {
    const body = { token };
    if (resource) body.resource = resource;
    if (action) body.action = action;
    const resp = await this._post('/auth/introspect', body);
    return resp.data ?? { active: false };
  }

  /**
   * 获取用户权限菜单 (GET)
   * @param {string} token
   * @returns {Promise<Object<string, string[]>>}
   */
  async getMenu(token) {
    const url = `${this.baseUrl}/auth/menu`;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
        signal: controller.signal,
      });
      const result = await response.json();
      if (result.code !== 0) throw new Error(result.message || '获取菜单失败');
      return result.data?.permissions ?? {};
    } finally {
      clearTimeout(timer);
    }
  }
}

/**
 * Express / Connect 权限校验中间件
 *
 * @param {RBACClient} client - RBAC 客户端实例
 * @param {string} resource - 资源名称
 * @param {string} action - 操作名称
 * @returns {Function} Express 中间件
 *
 * @example
 * const { RBACClient, permissionGuard } = require('rbac-client');
 * const client = new RBACClient('http://localhost:8087/api/v1');
 * app.delete('/users/:id', permissionGuard(client, 'user', 'delete'), handler);
 */
function permissionGuard(client, resource, action) {
  return async (req, res, next) => {
    const authHeader = req.headers.authorization || '';
    const token = authHeader.replace('Bearer ', '');

    if (!token) {
      return res.status(401).json({ code: 1002, message: '未提供认证Token' });
    }

    try {
      const allowed = await client.checkPermission(token, resource, action);
      if (!allowed) {
        return res.status(403).json({ code: 1003, message: '无权限' });
      }
      next();
    } catch (err) {
      return res.status(502).json({ code: 1005, message: '权限检查服务不可用' });
    }
  };
}

module.exports = { RBACClient, permissionGuard };
