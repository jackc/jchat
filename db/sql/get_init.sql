select row_to_json(t)
from (
  select
    (
      select coalesce(json_agg(row_to_json(t)), '[]'::json)
      from (
        select
          id,
          name,
          (
            select coalesce(json_agg(row_to_json(t)), '[]'::json)
            from (
              select
                id,
                user_id as author_id,
                body,
                extract(epoch from creation_time::timestamptz(0)) as creation_time
              from messages
              where messages.channel_id=channels.id
              order by creation_time asc
            ) t
          ) messages
        from channels
      ) t
    ) as channels,
    (
      select coalesce(json_agg(row_to_json(t)), '[]'::json)
      from (
        select id, name
        from users
        order by users.name
      ) t
    ) as users
) t
