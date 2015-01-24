select id, name, email, password_digest, password_salt
from users
where email=$1
