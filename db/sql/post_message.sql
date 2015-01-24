insert into messages(channel_id, user_id, body)
values($1, $2, $3)
returning id, creation_time
