-- Example Rule Script
-- Called for every incoming message
function on_message(gateway, data)
    print("Lua: Received message from " .. gateway .. ": " .. data)
    
    -- Filter specific messages
    if string.find(data, "DROP") then
        print("Lua: Dropping message containing DROP")
        return nil
    end

    -- Modify data
    if string.find(data, "HELLO") then
        return data .. " WORLD"
    end

    -- Pass through unchanged
    return data
end
