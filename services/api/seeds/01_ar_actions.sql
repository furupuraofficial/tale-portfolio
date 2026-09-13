-- Seed Data: AR Actions
-- Reusable AR actions that can be linked to any step

INSERT INTO ar_actions (id, type, target, params) VALUES
(1, 'PLAY_ANIMATION', 'zashiki', '{"name": "wave"}'),
(2, 'PLAY_ANIMATION', 'zashiki', '{"name": "bow"}'),
(3, 'PLAY_ANIMATION', 'zashiki', '{"name": "clap"}'),
(4, 'PLAY_ANIMATION', 'zashiki', '{"name": "explain"}'),
(5, 'PLAY_ANIMATION', 'zashiki', '{"name": "thumbs_up"}'),
(6, 'IDLE', 'zashiki', '{}'),
(7, 'SHOW_SPEECH_BUBBLE', 'zashiki', '{}'),
(8, 'HIDE_SPEECH_BUBBLE', 'zashiki', '{}'),
(9, 'MOVE_TO_NODE', 'entrance', '{}'),
(10, 'MOVE_TO_NODE', 'room_center', '{}'),
(11, 'SHOW_QUIZ', 'zashiki', '{"options": ["撮ってもらえませんか？", "撮ってくれませんか？", "撮りましょうか？"]}')
ON CONFLICT DO NOTHING;

-- Reset sequence to avoid conflicts on future inserts
SELECT setval('ar_actions_id_seq', (SELECT COALESCE(MAX(id), 0) FROM ar_actions));
