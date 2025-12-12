export interface Settings {
  credit_interval_minutes: number;
  credit_max: number;
  voting_paused: boolean;
  vote_visibility_mode: 'user_choice' | 'all_secret' | 'all_public';
  min_votes_for_ranking: number;
}

export interface UpdateSettingsRequest {
  credit_interval_minutes?: number;
  credit_max?: number;
  voting_paused?: boolean;
  vote_visibility_mode?: 'user_choice' | 'all_secret' | 'all_public';
  min_votes_for_ranking?: number;
}

export interface CreditActionResponse {
  message: string;
  users_affected: number;
}
