with t as (
  update password_resets
  set completion_ip=$1,
    completion_time=current_timestamp
  where token=$2
    and completion_time is null
  returning user_id
)
update users
set password_digest=$3,
  password_salt=$4
from t
where users.id=t.user_id
