package com.rbac.client;

/**
 * RBAC 服务返回的统一业务异常。
 */
public class RBACException extends Exception {

    private final int code;

    public RBACException(int code, String message) {
        super(message);
        this.code = code;
    }

    /** RBAC 业务错误码 */
    public int getCode() {
        return code;
    }

    /** 是否因为无权限 (code == 1003) */
    public boolean isForbidden() {
        return code == 1003;
    }

    /** 是否因为未认证 (code == 1002/1007/1008) */
    public boolean isUnauthorized() {
        return code == 1002 || code == 1007 || code == 1008;
    }
}
