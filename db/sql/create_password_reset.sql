insert into password_resets(token, user_id, request_ip, request_time)
values($1, $2, $3, current_timestamp)
