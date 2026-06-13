package com.rbac.client;

/**
 * RBAC 服务统一业务异常。
 */
public class RBACException extends Exception {

    private final int code;

    public RBACException(int code, String message) {
        super(message);
        this.code = code;
    }

    public int code() { return code; }
    public boolean isForbidden() { return code == 1003; }
    public boolean isUnauthorized() { return code == 1002 || code == 1007 || code == 1008; }
}
