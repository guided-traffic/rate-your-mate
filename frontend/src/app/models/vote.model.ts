import { Achievement } from './achievement.model';
import { User } from './user.model';

export interface Vote {
  id: number;
  from_user: User;
  to_user: User;
  achievement_id: string;
  achievement: Achievement;
  points: number;
  is_secret: boolean;
  created_at: string;
}

export interface CreateVoteRequest {
  to_user_id: number;
  achievement_id: string;
  points?: number; // 1-3 points, defaults to 1
  is_secret?: boolean; // null = use default (negative=secret, positive=open)
}

export interface VoteResponse {
  vote: Vote;
  credits: number;
}

export interface LeaderboardEntry {
  user: User;
  vote_count: number;
  rank: number;
}

export interface AchievementLeaderboard {
  achievement: Achievement;
  leaders: LeaderboardEntry[];
}

export interface Champion {
  user: User;
  achievement_count: number;
  total_votes: number;
}

export interface ChampionsResult {
  king: Champion | null;
  brother: Champion | null;
}
