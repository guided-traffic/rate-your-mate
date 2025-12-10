import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../environments/environment';
import { GamesResponse, Game } from '../models/game.model';

@Injectable({
  providedIn: 'root'
})
export class GameService {
  private readonly apiBase: string;

  constructor(private http: HttpClient) {
    // Extract base URL (e.g., 'http://localhost:8080' from 'http://localhost:8080/api/v1')
    this.apiBase = environment.apiUrl.replace(/\/api\/v1$/, '');
  }

  getMultiplayerGames(): Observable<GamesResponse> {
    return this.http.get<GamesResponse>(`${environment.apiUrl}/games`).pipe(
      map(response => this.resolveImageUrls(response))
    );
  }

  refreshGames(): Observable<GamesResponse> {
    return this.http.post<GamesResponse>(`${environment.apiUrl}/games/refresh`, {}).pipe(
      map(response => this.resolveImageUrls(response))
    );
  }

  /**
   * Resolves relative image URLs to full URLs
   */
  private resolveImageUrls(response: GamesResponse): GamesResponse {
    const resolveUrl = (url: string | undefined): string | undefined => {
      if (!url) return url;
      // If it's already an absolute URL, keep it
      if (url.startsWith('http://') || url.startsWith('https://')) {
        return url;
      }
      // Otherwise, prepend the API base
      return `${this.apiBase}${url}`;
    };

    const resolveGame = (game: Game): Game => ({
      ...game,
      header_image_url: resolveUrl(game.header_image_url) || game.header_image_url,
      capsule_image_url: resolveUrl(game.capsule_image_url) || game.capsule_image_url
    });

    return {
      ...response,
      pinned_games: response.pinned_games?.map(resolveGame) || [],
      all_games: response.all_games?.map(resolveGame) || []
    };
  }
}
