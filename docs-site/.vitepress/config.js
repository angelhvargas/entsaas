import { defineConfig } from 'vitepress'

export default defineConfig({
  lang: 'en-US',
  vite: {
    build: {
      target: 'esnext'
    }
  },
  title: 'EntSaaS Documentation',
  titleTemplate: ':title | EntSaaS Docs',
  description: 'Production-ready Go and Vue 3 framework for building secure, scalable multi-tenant SaaS platforms.',
  srcDir: 'src',
  outDir: '../dist/docs',
  cleanUrls: true,
  lastUpdated: true,
  metaChunk: true,
  ignoreDeadLinks: [
    /^https?:\/\/localhost/
  ],
  sitemap: {
    hostname: 'https://docs.entsaas.com'
  },
  
  transformHead: ({ pageData }) => {
    const head = []
    let urlPath = pageData.relativePath
      .replace(/index\.md$/, '')
      .replace(/\.md$/, '')
    
    const canonicalUrl = `https://docs.entsaas.com/${urlPath}`
    head.push(['link', { rel: 'canonical', href: canonicalUrl }])
    return head
  },

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/favicon-32x32.png' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '16x16', href: '/favicon-16x16.png' }],
    ['link', { rel: 'apple-touch-icon', sizes: '180x180', href: '/apple-touch-icon.png' }],
    ['meta', { name: 'theme-color', content: '#6366F1' }],
    ['meta', { name: 'description', content: 'Production-ready Go and Vue 3 framework for building secure, scalable multi-tenant SaaS platforms.' }],
    ['meta', { property: 'og:title', content: 'EntSaaS Documentation' }],
    ['meta', { property: 'og:description', content: 'High-performance multi-tenant Go + Vue 3 framework. Complete with JWT, RBAC, Stripe/Paddle billing, and AI SSE streaming.' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    ['link', { href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap', rel: 'stylesheet' }],
  ],

  themeConfig: {
    logo: {
      light: '/entsaas-logo-positive.svg',
      dark: '/entsaas-logo-negative.svg',
      alt: 'EntSaaS Documentation Home'
    },
    siteTitle: false,
    nav: [
      { text: 'Guide', link: '/guide/', activeMatch: '/guide/', ariaLabel: 'Guide section navigation' },
      { text: 'Architecture', link: '/architecture/', activeMatch: '/architecture/', ariaLabel: 'Architecture deep-dive documentation' },
      { text: 'API Reference', link: '/api/', activeMatch: '/api/', ariaLabel: 'API Reference documentation' },
      { text: 'Support', link: '/support/', activeMatch: '/support/', ariaLabel: 'Support and contact information' }
    ],
    sidebar: {
      '/guide/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Introduction', link: '/guide/' },
            { text: 'Quickstart', link: '/guide/quickstart' },
            { text: 'Installation', link: '/guide/installation' }
          ]
        },
        {
          text: 'Operations',
          items: [
            { text: 'CLI Administration', link: '/guide/cli' }
          ]
        }
      ],
      '/architecture/': [
        {
          text: 'Core Modules',
          items: [
            { text: 'System Overview', link: '/architecture/' },
            { text: 'Multi-Tenancy', link: '/architecture/multi-tenancy' },
            { text: 'Authentication & RBAC', link: '/architecture/auth' },
            { text: 'Billing & Catalog', link: '/architecture/billing' },
            { text: 'AI Integration', link: '/architecture/ai-integration' }
          ]
        }
      ],
      '/api/': [
        {
          text: 'API Reference',
          items: [
            { text: 'Overview', link: '/api/' }
          ]
        }
      ],
      '/support/': [
        {
          text: 'Support',
          items: [
            { text: 'Contact & Help', link: '/support/' }
          ]
        }
      ]
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/angelhvargas/entsaas' }
    ],
    footer: {
      message: 'Released under the Commercial License. <a href="https://entsaas.com/privacy">Privacy &amp; Cookies</a>',
      copyright: 'Copyright © 2026 EntSaaS'
    },
    search: {
      provider: 'local'
    },
    editLink: {
      pattern: 'https://github.com/angelhvargas/entsaas/edit/master/docs-site/src/:path',
      text: 'Suggest changes to this page'
    }
  }
})
