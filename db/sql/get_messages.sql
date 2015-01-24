select id, user_id, body, creation_time
from messages
where channel_id=$1
order by id desc
