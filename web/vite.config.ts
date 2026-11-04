import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { VitePWA } from 'vite-plugin-pwa';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		VitePWA({
			registerType: 'autoUpdate',
			manifest: {
				name: 'Integrated Personal Budget',
				short_name: 'Budget',
				description: 'Personal budgeting, transactions, and spending insights.',
				theme_color: '#0f172a',
				background_color: '#ffffff',
				display: 'standalone',
				start_url: '/',
				icons: [
					{ src: '/favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' }
				]
			},
			workbox: {
				globPatterns: ['**/*.{js,css,html,svg,png,ico}'],
				navigateFallback: '/index.html'
			}
		})
	],
	server: {
		proxy: {
			'/api': 'http://127.0.0.1:8080'
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		passWithNoTests: true
	}
});
