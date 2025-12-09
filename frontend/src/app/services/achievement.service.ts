import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map, tap } from 'rxjs';
import { environment } from '../../environments/environment';
import { Achievement, AchievementsResponse } from '../models/achievement.model';

@Injectable({
  providedIn: 'root'
})
export class AchievementService {
  private achievementsCache = signal<Map<string, Achievement>>(new Map());

  constructor(private http: HttpClient) {}

  getAll(): Observable<AchievementsResponse> {
    return this.http.get<AchievementsResponse>(`${environment.apiUrl}/achievements`)
      .pipe(
        tap(response => {
          // Cache all achievements
          const cache = new Map<string, Achievement>();
          response.achievements.forEach(a => cache.set(a.id, a));
          this.achievementsCache.set(cache);
        })
      );
  }

  getById(id: string): Observable<Achievement> {
    return this.http.get<{ achievement: Achievement }>(`${environment.apiUrl}/achievements/${id}`)
      .pipe(map(response => response.achievement));
  }

  /**
   * Get achievement description from cache
   * Returns empty string if not found
   */
  getDescription(id: string): string {
    return this.achievementsCache().get(id)?.description || '';
  }

  /**
   * Get achievement from cache
   */
  getCached(id: string): Achievement | undefined {
    return this.achievementsCache().get(id);
  }

  /**
   * Load achievements into cache (call on app init)
   */
  loadCache(): Observable<AchievementsResponse> {
    return this.getAll();
  }
}
