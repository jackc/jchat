insert into users(name, email, password_digest, password_salt)
values($1, $2, $3, $4)
returning id, name, email
