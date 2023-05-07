select "user", count(*) as video_count, avg(audio_duration) as avg_audio_duration_seconds
from transcriptions
where has_error = 0
group by "user";

select id, user, last_conversion_time, mp3_file_name, audio_duration, transcription, error_message
from transcriptions
where 1=1
  and has_error = 0
  and "user" = '安教授_'
  --and last_conversion_time > '2023-04-30 12:43:15.664385+08:00'
order by last_conversion_time desc;

select *
from transcriptions
where 1=1
  and has_error = 0
  and "user" = '薛辉小清新'
  and transcription like '%400万%'
order by last_conversion_time desc;

-- update transcriptions set user = '刘永丰'
-- where 1=1
--   and has_error = 0
--   and user = '汪雪芬_一诺'
-- and last_conversion_time > '2023-04-30 12:43:15.664385+08:00';

SELECT sql FROM sqlite_master WHERE type='table' AND name='transcriptions';
