if redis.call("EXISTS", KEYS[1]) == 0 then
    redis.call("HSET", KEYS[1], unpack(ARGV))
    return "ok"
else
    return ""
end
