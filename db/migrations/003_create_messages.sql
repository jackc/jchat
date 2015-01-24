create table messages(
  id bigserial primary key,
  channel_id integer not null references channels,
  creation_time timestamptz not null default now(),
  user_id integer not null references users,
  body text not null
);

create index on messages (channel_id);
create index on messages (user_id);

grant select, insert, update, delete on messages to {{.app_user}};
grant usage on sequence messages_id_seq to {{.app_user}};

---- create above / drop below ----

drop table messages;
