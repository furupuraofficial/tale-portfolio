-- Seed Data: Lessons
-- Run this after ar_actions.sql

-- Lessons
INSERT INTO lessons (id, title, description) VALUES
('001', 'Say "Delicious" at a Restaurant', 'Learn two ways to express "delicious" in Japanese'),
('002', 'Ask for Recommendations', 'Learn how to ask staff for recommended items'),
('003', 'Ask Someone to Take a Photo', 'Learn how to politely ask someone to take your picture')
ON CONFLICT (id) DO NOTHING;

-- Lesson 1: Say "Delicious" at a Restaurant
-- ar_action_id references: 1=wave, 4=explain, 6=idle, 7=speech_bubble, 3=clap, 5=thumbs_up
INSERT INTO lesson_steps (lesson_id, step_id, step_name, content, mic_enabled_after, step_order, ar_action_id) VALUES
('001', 'L1_INTRO', 'Introduction', 'Did you know there are two ways to say "delicious" in Japanese?', false, 1, 1),
('001', 'L1_OISHII', 'Oishii', 'The first way is "Oishii". This is the most common way to say delicious!', false, 2, 4),
('001', 'L1_OISHII_PRACTICE', 'Practice Oishii', 'Now try saying "Oishii" yourself!', true, 3, 7),
('001', 'L1_UMAI', 'Umai', 'The second way is "Umai". This is more casual and often used by men.', false, 4, 4),
('001', 'L1_UMAI_PRACTICE', 'Practice Umai', 'Now try saying "Umai"!', true, 5, 7),
('001', 'L1_COMPLETE', 'Lesson Complete', 'Great job! Now you know two ways to say delicious: "Oishii" for polite situations, and "Umai" for casual ones!', false, 6, 5)
ON CONFLICT (lesson_id, step_id) DO NOTHING;

-- Lesson 2: Ask for Recommendations
INSERT INTO lesson_steps (lesson_id, step_id, step_name, content, mic_enabled_after, step_order, ar_action_id) VALUES
('002', 'L2_INTRO', 'Introduction', 'Today we will learn how to ask staff for their recommendations!', false, 1, 1),
('002', 'L2_OSUSUME', 'Osusume', 'To ask "What do you recommend?", say "Osusume wa nan desu ka?"', false, 2, 4),
('002', 'L2_OSUSUME_PRACTICE', 'Practice', 'Try asking "Osusume wa nan desu ka?"', true, 3, 7),
('002', 'L2_NINKI', 'Popular Items', 'You can also ask "What is popular?" by saying "Ninki wa nan desu ka?"', false, 4, 4),
('002', 'L2_NINKI_PRACTICE', 'Practice', 'Try asking "Ninki wa nan desu ka?"', true, 5, 7),
('002', 'L2_COMPLETE', 'Lesson Complete', 'Excellent! Now you can ask for recommendations using "Osusume" or ask about popular items using "Ninki"!', false, 6, 5)
ON CONFLICT (lesson_id, step_id) DO NOTHING;

-- Lesson 3: Ask Someone to Take a Photo
-- ar_action_id references: 1=wave, 4=explain, 6=idle, 7=speech_bubble, 3=clap, 5=thumbs_up, 11=quiz
INSERT INTO lesson_steps (lesson_id, step_id, step_name, content, mic_enabled_after, step_order, ar_action_id) VALUES
('003', 'L3_INTRO', 'Introduction', 'Today we will learn how to ask someone to take your picture!', false, 1, 1),
('003', 'L3_PHRASE', 'Phrase Introduction', 'To politely ask "Could you take a picture of me?", say "Shashin o totte kuremsen ka?"', false, 2, 4),
('003', 'L3_MEANING', 'Meaning Explanation', '"Shashin" means photo, "totte" means take, and "kuremsen ka" is a polite way to ask for a favor.', false, 3, 4),
('003', 'L3_PRONUNCIATION', 'Pronunciation Guide', 'Listen carefully: SHA-SHIN O TOT-TE KU-RE-MA-SEN KA. The key is the polite ending "kuremsen ka".', false, 4, 6),
('003', 'L3_PRACTICE', 'Voice Practice', 'Now try saying "Shashin o totte kuremsen ka?" yourself!', true, 5, 7),
('003', 'L3_EXTENDED', 'Extended Practice', 'Great! Now try saying it a bit faster and more naturally.', true, 6, 7),
('003', 'L3_QUIZ', 'Quiz', 'Quiz time! Which phrase means "Could you take a picture?"', false, 7, 11),
('003', 'L3_COMPLETE', 'Lesson Complete', 'Congratulations! You have learned how to ask someone to take your picture in Japanese!', false, 8, 5)
ON CONFLICT (lesson_id, step_id) DO NOTHING;
