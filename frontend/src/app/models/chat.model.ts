import { User } from './user.model';

export interface AchievementBadge {
  id: string;
  name: string;
  description: string;
  image_url: string;
  is_positive: boolean;
  count: number;
}

export interface ChatMessage {
  id: number;
  user: User;
  message: string;
  achievements: AchievementBadge[];
  created_at: string;
}

export interface CreateChatMessageRequest {
  message: string;
}

export interface ChatMessagesResponse {
  messages: ChatMessage[];
}
