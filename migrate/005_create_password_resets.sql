create table password_resets(
  token varchar primary key,
  user_id integer not null references users,
  request_ip inet not null,
  request_time timestamptz not null,
  completion_ip inet,
  completion_time timestamptz,
  check(completion_ip is null = completion_time is null)
);

grant select, insert, update, delete on password_resets to {{.app_user}};

---- create above / drop below ----

drop table password_resets;
