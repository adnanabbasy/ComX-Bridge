# Script Plugin Development Guide

ComX-Bridge supports lightweight plugins written in Lua or JavaScript (ECMAScript 5.1).

## Supported Languages

| Language | Engine | Extension | Use Case |
|----------|--------|-----------|----------|
| **Lua** | gopher-lua | `.lua` | High performance, embedded logic |
| **JavaScript** | goja | `.js` | Web-like parsing logic |

## Lua Plugin Example

```lua
-- parser.lua
function Parse(data)
    -- Input: byte array (userdata or string)
    -- Output: table with packet info
    
    local str = tostring(data)
    if string.sub(str, 1, 1) == "$" then
        return {
            valid = true,
            payload = string.sub(str, 2)
        }
    end
    
    return { valid = false }
end
```

## JavaScript Plugin Example

```javascript
// parser.js
function parse(data) {
    // data is a byte array
    var str = String.fromCharCode.apply(null, data);
    
    if (str.startsWith("HEAD")) {
        return {
            type: "header_packet",
            content: str.substring(4)
        };
    }
    
    return null;
}
```

## Deployment
Place script files in the `plugins/scripts/` directory. They are automatically loaded on startup if `ai.features.code_generation` or script loader is enabled.
