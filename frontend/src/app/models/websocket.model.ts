export type WebSocketMessageType = 'vote_received' | 'new_vote' | 'user_joined' | 'settings_update' | 'credits_reset' | 'credits_given' | 'chat_message' | 'new_king' | 'games_sync_progress' | 'games_sync_complete' | 'vote_invalidation' | 'error';

export interface WebSocketMessage<T = unknown> {
  type: WebSocketMessageType;
  payload: T;
}

export interface VotePayload {
  vote_id: number;
  from_user_id: number;
  from_username: string;
  from_avatar: string;
  to_user_id: number;
  to_username: string;
  to_avatar: string;
  achievement_id: string;
  achievement_name: string;
  is_positive: boolean;
  is_secret: boolean;
  created_at: string;
}

export interface SettingsPayload {
  credit_interval_minutes: number;
  credit_max: number;
  voting_paused: boolean;
  vote_visibility_mode: 'user_choice' | 'all_secret' | 'all_public';
  negative_voting_disabled: boolean;
  countdown_target?: string | null;
}

export interface CreditActionPayload {
  message: string;
}

export interface NewKingPayload {
  user_id: number;
  username: string;
  avatar: string;
}

export interface ChatMessagePayload {
  id: number;
  user_id: number;
  username: string;
  steam_id: string;
  avatar_small: string;
  message: string;
  achievements: Array<{
    id: string;
    name: string;
    description: string;
    image_url: string;
    is_positive: boolean;
    count: number;
  }>;
  created_at: string;
}

export interface GamesSyncProgressPayload {
  phase: 'fetching_users' | 'fetching_categories' | 'complete';
  current_game: string;
  processed_count: number;
  total_count: number;
  percentage: number;
}

export interface GamesSyncCompletePayload {
  message: string;
  total_games: number;
}

export interface VoteInvalidationPayload {
  vote_id: number;
  is_invalidated: boolean;
}
