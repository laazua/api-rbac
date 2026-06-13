package com.rbac.client;

import java.util.*;

/**
 * 极简 JSON 解析器 — 零依赖实现。
 *
 * <p>仅用于 SDK 内部解析 api-rbac 的 JSON 响应，支持对象、数组、字符串、数字、布尔、null。
 * 不为通用场景设计，省略了转义处理等边缘情况（api-rbac 响应的 JSON 是程序生成的，格式固定）。</p>
 */
class JsonNode {

    private final Object value;

    private JsonNode(Object value) { this.value = value; }

    // ---- 工厂方法 ----

    @SuppressWarnings("unchecked")
    static JsonNode parse(String json) {
        Parser p = new Parser(json.trim());
        Object val = p.parseValue();
        return new JsonNode(val);
    }

    @SuppressWarnings("unchecked")
    static String toJson(Object obj) {
        StringBuilder sb = new StringBuilder();
        writeValue(sb, obj);
        return sb.toString();
    }

    // ---- 类型获取 ----

    public int getInt(String key)     { return ((Number) getObj(key)).intValue(); }
    public long getLong(String key)   { return ((Number) getObj(key)).longValue(); }
    public boolean getBoolean(String key) { return (Boolean) getObj(key); }
    public String getString(String key)   { return (String) getObj(key); }

    public JsonNode getObject(String key) {
        Object v = getObj(key);
        return v == null ? new JsonNode(null) : new JsonNode(v);
    }

    @SuppressWarnings("unchecked")
    public List<JsonNode> getArray(String key) {
        List<Object> arr = (List<Object>) getObj(key);
        if (arr == null) return Collections.emptyList();
        List<JsonNode> result = new ArrayList<>(arr.size());
        for (Object o : arr) result.add(new JsonNode(o));
        return result;
    }

    @SuppressWarnings("unchecked")
    public Set<String> keys() {
        if (value instanceof Map) return ((Map<String, Object>) value).keySet();
        return Collections.emptySet();
    }

    // 无 key 直接取值
    public int    asInt()    { return ((Number) value).intValue(); }
    public long   asLong()   { return ((Number) value).longValue(); }
    public String asString() { return (String) value; }
    public boolean asBoolean() { return (Boolean) value; }

    // opt 版本 (可为 null)
    public long optLong(String key, long def) {
        Object v = opt(key); return v instanceof Number ? ((Number) v).longValue() : def;
    }
    public String optString(String key, String def) {
        Object v = opt(key); return v instanceof String ? (String) v : def;
    }

    @SuppressWarnings("unchecked")
    private Object getObj(String key) {
        if (value instanceof Map) return ((Map<String, Object>) value).get(key);
        return null;
    }

    @SuppressWarnings("unchecked")
    private Object opt(String key) {
        if (value instanceof Map) return ((Map<String, Object>) value).get(key);
        return null;
    }

    // ---- 内部序列化 ----

    @SuppressWarnings("unchecked")
    private static void writeValue(StringBuilder sb, Object val) {
        if (val == null) { sb.append("null"); return; }
        if (val instanceof String) {
            sb.append('"').append(val).append('"');
        } else if (val instanceof Number || val instanceof Boolean) {
            sb.append(val);
        } else if (val instanceof Map) {
            sb.append('{');
            boolean first = true;
            for (Map.Entry<String, Object> e : ((Map<String, Object>) val).entrySet()) {
                if (!first) sb.append(',');
                sb.append('"').append(e.getKey()).append("\":");
                writeValue(sb, e.getValue());
                first = false;
            }
            sb.append('}');
        } else if (val instanceof List) {
            sb.append('[');
            boolean first = true;
            for (Object item : (List<Object>) val) {
                if (!first) sb.append(',');
                writeValue(sb, item);
                first = false;
            }
            sb.append(']');
        }
    }

    // ---- 内部 JSON 解析器 ----

    private static class Parser {
        private final String json;
        private int pos;

        Parser(String json) { this.json = json; this.pos = 0; }

        Object parseValue() {
            skipWs();
            if (pos >= json.length()) return null;
            char c = json.charAt(pos);
            if (c == '{') return parseObject();
            if (c == '[') return parseArray();
            if (c == '"') return parseString();
            if (c == 't' || c == 'f') return parseBoolean();
            if (c == 'n') return parseNull();
            return parseNumber();
        }

        Map<String, Object> parseObject() {
            Map<String, Object> map = new LinkedHashMap<>();
            expect('{');
            skipWs();
            if (json.charAt(pos) == '}') { pos++; return map; }
            while (true) {
                skipWs();
                String key = parseString();
                skipWs();
                expect(':');
                skipWs();
                Object val = parseValue();
                map.put(key, val);
                skipWs();
                if (json.charAt(pos) == '}') { pos++; return map; }
                expect(',');
            }
        }

        List<Object> parseArray() {
            List<Object> list = new ArrayList<>();
            expect('[');
            skipWs();
            if (json.charAt(pos) == ']') { pos++; return list; }
            while (true) {
                skipWs();
                list.add(parseValue());
                skipWs();
                if (json.charAt(pos) == ']') { pos++; return list; }
                expect(',');
            }
        }

        String parseString() {
            expect('"');
            StringBuilder sb = new StringBuilder();
            while (pos < json.length()) {
                char c = json.charAt(pos++);
                if (c == '"') return sb.toString();
                if (c == '\\') {
                    char next = json.charAt(pos++);
                    if (next == '"') sb.append('"');
                    else if (next == '\\') sb.append('\\');
                    else if (next == 'n') sb.append('\n');
                    else if (next == 't') sb.append('\t');
                    else sb.append(next);
                } else {
                    sb.append(c);
                }
            }
            throw new RuntimeException("Unterminated string");
        }

        Number parseNumber() {
            int start = pos;
            while (pos < json.length() && (Character.isDigit(json.charAt(pos))
                    || json.charAt(pos) == '.' || json.charAt(pos) == '-' || json.charAt(pos) == 'e'
                    || json.charAt(pos) == 'E' || json.charAt(pos) == '+')) {
                pos++;
            }
            String num = json.substring(start, pos);
            if (num.contains(".")) return Double.parseDouble(num);
            long l = Long.parseLong(num);
            if (l >= Integer.MIN_VALUE && l <= Integer.MAX_VALUE) return (int) l;
            return l;
        }

        Boolean parseBoolean() {
            if (json.startsWith("true", pos)) { pos += 4; return true; }
            pos += 5; return false;
        }

        Object parseNull() { pos += 4; return null; }

        void expect(char c) {
            skipWs();
            if (json.charAt(pos) != c)
                throw new RuntimeException("Expected '" + c + "' at pos " + pos + " but got '" + json.charAt(pos) + "'");
            pos++;
        }

        void skipWs() {
            while (pos < json.length() && Character.isWhitespace(json.charAt(pos))) pos++;
        }
    }
}
