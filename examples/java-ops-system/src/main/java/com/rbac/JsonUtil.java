package com.rbac;

import java.util.*;

/**
 * 极简 JSON 工具 — 无外部依赖，仅用于 api-rbac 的规整 JSON 响应。
 */
final class JsonUtil {

    /** 解析 JSON 字符串为 Map / List / String / Number / Boolean / null */
    static Object parse(String json) {
        return new Parser(json.trim()).parseValue();
    }

    /** 将 Map/List 序列化为 JSON 字符串 */
    @SuppressWarnings("unchecked")
    static String toJson(Object obj) {
        var sb = new StringBuilder();
        writeValue(sb, obj);
        return sb.toString();
    }

    @SuppressWarnings("unchecked")
    private static void writeValue(StringBuilder sb, Object val) {
        if (val == null) {
            sb.append("null");
        } else if (val instanceof String s) {
            sb.append('"').append(escape(s)).append('"');
        } else if (val instanceof Number n) {
            sb.append(n);
        } else if (val instanceof Boolean b) {
            sb.append(b);
        } else if (val instanceof Map m) {
            sb.append('{');
            boolean first = true;
            for (Object e : m.entrySet()) {
                Map.Entry<?, ?> entry = (Map.Entry<?, ?>) e;
                if (!first) sb.append(',');
                sb.append('"').append(escape(entry.getKey().toString())).append("\":");
                writeValue(sb, entry.getValue());
                first = false;
            }
            sb.append('}');
        } else if (val instanceof List l) {
            sb.append('[');
            boolean first = true;
            for (var item : l) {
                if (!first) sb.append(',');
                writeValue(sb, item);
                first = false;
            }
            sb.append(']');
        } else {
            sb.append('"').append(escape(val.toString())).append('"');
        }
    }

    private static String escape(String s) {
        return s.replace("\\", "\\\\").replace("\"", "\\\"")
                .replace("\n", "\\n").replace("\t", "\\t");
    }

    // ---- 解析器 ----
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
            var map = new LinkedHashMap<String, Object>();
            expect('{');
            skipWs();
            if (pos < json.length() && json.charAt(pos) == '}') { pos++; return map; }
            while (true) {
                skipWs();
                String key = parseString();
                skipWs();
                expect(':');
                skipWs();
                map.put(key, parseValue());
                skipWs();
                if (pos < json.length() && json.charAt(pos) == '}') { pos++; return map; }
                expect(',');
            }
        }

        List<Object> parseArray() {
            var list = new ArrayList<>();
            expect('[');
            skipWs();
            if (pos < json.length() && json.charAt(pos) == ']') { pos++; return list; }
            while (true) {
                skipWs();
                list.add(parseValue());
                skipWs();
                if (pos < json.length() && json.charAt(pos) == ']') { pos++; return list; }
                expect(',');
            }
        }

        String parseString() {
            expect('"');
            var sb = new StringBuilder();
            while (pos < json.length()) {
                char c = json.charAt(pos++);
                if (c == '"') return sb.toString();
                if (c == '\\') {
                    char esc = json.charAt(pos++);
                    if (esc == '"') sb.append('"');
                    else if (esc == '\\') sb.append('\\');
                    else if (esc == 'n') sb.append('\n');
                    else if (esc == 't') sb.append('\t');
                    else sb.append(esc);
                } else sb.append(c);
            }
            throw new RuntimeException("Unterminated string");
        }

        Number parseNumber() {
            int start = pos;
            while (pos < json.length()) {
                char c = json.charAt(pos);
                if (Character.isDigit(c) || c == '.' || c == '-' || c == 'e' || c == 'E' || c == '+') pos++;
                else break;
            }
            String num = json.substring(start, pos);
            return num.contains(".") ? Double.parseDouble(num) : Long.parseLong(num);
        }

        Boolean parseBoolean() {
            if (json.startsWith("true", pos)) { pos += 4; return true; }
            pos += 5; return false;
        }

        Object parseNull() { pos += 4; return null; }

        void expect(char c) {
            skipWs();
            if (pos >= json.length() || json.charAt(pos) != c)
                throw new RuntimeException("Expected '" + c + "' at pos " + pos);
            pos++;
        }

        void skipWs() {
            while (pos < json.length() && Character.isWhitespace(json.charAt(pos))) pos++;
        }
    }
}
