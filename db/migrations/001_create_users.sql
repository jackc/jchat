create table users(
  id serial primary key,
  email varchar not null,
  name varchar(30) not null check(name ~ '\A[a-zA-Z0-9]+\Z'),
  password_digest bytea not null,
  password_salt bytea not null,
  creation_time timestamptz not null default now()
);

create unique index users_email_unq on users (lower(email));
create unique index users_name_unq on users (lower(name));

grant select, insert, update, delete on users to {{.app_user}};
grant usage on sequence users_id_seq to {{.app_user}};

---- create above / drop below ----

drop table users;
