DELETE FROM public.read_model_versions
WHERE owner_user_id = 178220800
  AND peer_type = 'user'
  AND peer_id = 178220800
  AND model IN ('contact_account', 'channel_active_memberships');

DELETE FROM public.peer_usernames
WHERE username_lower = 'spambot'
  AND peer_type = 'user'
  AND peer_id = 178220800;

DELETE FROM public.bots
WHERE bot_user_id = 178220800
  AND owner_user_id = 178220800;

DELETE FROM public.users
WHERE id = 178220800
  AND username = 'SpamBot'
  AND is_bot = true;
