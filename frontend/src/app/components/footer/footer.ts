import { Component, inject } from '@angular/core';
import { VersionService } from '../../services/version.service';

@Component({
  selector: 'app-footer',
  standalone: true,
  templateUrl: './footer.html',
  styleUrl: './footer.scss'
})
export class FooterComponent {
  currentYear = new Date().getFullYear();
  versionService = inject(VersionService);
}
