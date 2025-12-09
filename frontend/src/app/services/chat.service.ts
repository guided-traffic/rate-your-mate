import { Injectable, signal, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../environments/environment';
import { ChatMessage, ChatMessagesResponse, CreateChatMessageRequest } from '../models/chat.model';
import { ChatMessagePayload } from '../models/websocket.model';
import { WebSocketService } from './websocket.service';

@Injectable({
  providedIn: 'root'
})
export class ChatService {
  private http = inject(HttpClient);
  private wsService = inject(WebSocketService);

  private messages = signal<ChatMessage[]>([]);
  readonly chatMessages = this.messages.asReadonly();

  constructor() {
    // Subscribe to WebSocket chat messages
    this.wsService.chatMessage$.subscribe((payload) => {
      this.addMessageFromPayload(payload);
    });
  }

  loadMessages(): Observable<ChatMessage[]> {
    return this.http.get<ChatMessagesResponse>(`${environment.apiUrl}/chat`)
      .pipe(
        map(response => {
          const msgs = response.messages || [];
          this.messages.set(msgs);
          return msgs;
        })
      );
  }

  sendMessage(message: string): Observable<{ message: ChatMessage }> {
    const request: CreateChatMessageRequest = { message };
    return this.http.post<{ message: ChatMessage }>(`${environment.apiUrl}/chat`, request);
  }

  private addMessageFromPayload(payload: ChatMessagePayload): void {
    // Convert payload to ChatMessage format
    const newMessage: ChatMessage = {
      id: payload.id,
      user: {
        id: payload.user_id,
        steam_id: payload.steam_id,
        username: payload.username,
        avatar_url: payload.avatar_small || '',
        avatar_small: payload.avatar_small || '',
        profile_url: ''
      },
      message: payload.message,
      achievements: payload.achievements || [],
      created_at: payload.created_at
    };

    // Add to messages array
    this.messages.update(msgs => [...msgs, newMessage]);
  }

  clearMessages(): void {
    this.messages.set([]);
  }
}
