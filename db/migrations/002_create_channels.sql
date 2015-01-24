create table channels(
  id serial primary key,
  name varchar(30) not null check(name ~ '\A[a-zA-Z0-9]+\Z'),
  creation_time timestamptz not null default now()
);

create unique index channels_name_unq on channels (lower(name));

grant select, insert, update, delete on channels to {{.app_user}};
grant usage on sequence channels_id_seq to {{.app_user}};

---- create above / drop below ----

drop table channels;
