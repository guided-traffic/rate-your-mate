export type WebSocketMessageType = 'vote_received' | 'new_vote' | 'user_joined' | 'settings_update' | 'credits_reset' | 'credits_given' | 'chat_message' | 'error';

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
  created_at: string;
}

export interface SettingsPayload {
  credit_interval_minutes: number;
  credit_max: number;
  voting_paused: boolean;
}

export interface CreditActionPayload {
  message: string;
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
