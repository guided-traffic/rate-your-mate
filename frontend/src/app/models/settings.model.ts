export interface Settings {
  credit_interval_minutes: number;
  credit_max: number;
  voting_paused: boolean;
  vote_visibility_mode: 'user_choice' | 'all_secret' | 'all_public';
  min_votes_for_ranking: number;
  negative_voting_disabled: boolean;
  countdown_target?: string | null; // RFC3339 formatted time, null if not set
}

export interface UpdateSettingsRequest {
  credit_interval_minutes?: number;
  credit_max?: number;
  voting_paused?: boolean;
  vote_visibility_mode?: 'user_choice' | 'all_secret' | 'all_public';
  min_votes_for_ranking?: number;
  negative_voting_disabled?: boolean;
  countdown_target?: string | null; // RFC3339 formatted time, empty string or null to clear
}

export interface CreditActionResponse {
  message: string;
  users_affected: number;
}
