import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../environments/environment';

export interface HealthResponse {
  status: string;
  version: string;
  buildTime: string;
  gitCommit: string;
}

@Injectable({
  providedIn: 'root'
})
export class VersionService {
  // Frontend version - from environment (injected at build time)
  readonly frontendVersion = signal<string>(environment.version);

  // Backend version - fetched from health endpoint
  readonly backendVersion = signal<string>('...');
  readonly backendBuildTime = signal<string>('');
  readonly backendGitCommit = signal<string>('');

  private readonly healthUrl: string;

  constructor(private http: HttpClient) {
    // Health endpoint is at root level, not under /api/v1
    this.healthUrl = environment.apiUrl.replace('/api/v1', '') + '/health';
    this.loadBackendVersion();
  }

  private loadBackendVersion(): void {
    this.http.get<HealthResponse>(this.healthUrl).subscribe({
      next: (response) => {
        this.backendVersion.set(response.version || 'unknown');
        this.backendBuildTime.set(response.buildTime || '');
        this.backendGitCommit.set(response.gitCommit || '');
      },
      error: () => {
        this.backendVersion.set('offline');
      }
    });
  }

  refresh(): void {
    this.loadBackendVersion();
  }
}
