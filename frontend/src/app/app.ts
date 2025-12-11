import { Component, inject, OnInit, OnDestroy, effect } from '@angular/core';
import { RouterOutlet, RouterLink } from '@angular/router';
import { CommonModule } from '@angular/common';
import { HeaderComponent } from './components/header/header.component';
import { FooterComponent } from './components/footer/footer';
import { NotificationsComponent } from './components/notifications/notifications.component';
import { LoadingOverlayComponent } from './components/loading-overlay/loading-overlay.component';
import { AuthService } from './services/auth.service';
import { WebSocketService } from './services/websocket.service';
import { ConnectionStatusService } from './services/connection-status.service';
import { AchievementService } from './services/achievement.service';
import { ChatService } from './services/chat.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, HeaderComponent, FooterComponent, NotificationsComponent, LoadingOverlayComponent],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnInit, OnDestroy {
  private authService = inject(AuthService);
  private wsService = inject(WebSocketService);
  private connectionStatus = inject(ConnectionStatusService);
  private achievementService = inject(AchievementService);
  chatService = inject(ChatService);

  get isAuthenticated(): boolean {
    return this.authService.isAuthenticated();
  }

  constructor() {
    // Connect/disconnect WebSocket based on authentication state
    effect(() => {
      const isAuth = this.authService.isAuthenticated();
      const user = this.authService.user();
      const isLoading = this.authService.isLoading();

      if (isAuth && user) {
        // User is authenticated and data is loaded
        this.wsService.connect();
        this.connectionStatus.markInitialLoadComplete();
      } else if (isAuth && !user && !isLoading) {
        // Has token but user load failed (e.g., network error) - still complete initial load
        // The reconnect logic will handle recovery
        this.connectionStatus.markInitialLoadComplete();
      } else if (!isAuth) {
        // Not authenticated - show login page
        this.wsService.disconnect();
        this.connectionStatus.markInitialLoadComplete();
      }
      // If isAuth && !user && isLoading, we're still loading - don't mark complete yet
    });
  }

  ngOnInit(): void {
    // Load achievements cache for tooltips in chat
    this.achievementService.loadCache();
  }

  ngOnDestroy(): void {
    this.wsService.disconnect();
  }
}
