update users
set password_digest=$1,
  password_salt=$2
where id=$3
