const config = {
  // Disable server-side rendering: https://go.nuxtjs.dev/ssr-mode
  ssr: false,

  // Global page headers: https://go.nuxtjs.dev/config-head
  head: {
    title: 'ISUCHOLAR',
    htmlAttrs: {
      lang: 'ja',
    },
    meta: [
      { charset: 'utf-8' },
      { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      { hid: 'description', name: 'description', content: '' },
    ],
    link: [{ rel: 'icon', color: 'image/png', href: '/favicon.png' }],
  },

  // Global CSS: https://go.nuxtjs.dev/config-css
  css: ['@/assets/css/style.css'],

  // Plugins to run before rendering page: https://go.nuxtjs.dev/config-plugins
  plugins: ['~/plugins/axios'],

  // Auto import components: https://go.nuxtjs.dev/config-components
  components: ['~/components/', '~/components/common'],

  // Modules for dev and build (recommended): https://go.nuxtjs.dev/config-modules
  buildModules: [
    // https://go.nuxtjs.dev/typescript
    '@nuxt/typescript-build',
    // https://go.nuxtjs.dev/stylelint
    '@nuxtjs/stylelint-module',
    // https://go.nuxtjs.dev/tailwindcss
    '@nuxtjs/tailwindcss',
    [
      '@nuxtjs/fontawesome',
      {
        component: 'Fa',
        suffix: true,
        icons: {
          solid: [
            'faTimes',
            'faChevronLeft',
            'faChevronRight',
            'faPen',
            'faPlus',
            'faBell',
            'faGraduationCap',
            'faEllipsisV',
            'faInfoCircle',
            'faExclamationTriangle',
          ],
          regular: ['faClock'],
        },
      },
    ],
  ],

  // Modules: https://go.nuxtjs.dev/config-modules
  modules: ['@nuxtjs/proxy', 'cookie-universal-nuxt'],

  // Build Configuration: https://go.nuxtjs.dev/config-build
  build: {
    filenames: {
      app: ({ isModern }) => `[name]${isModern ? '.modern' : ''}.js`,
      chunk: ({ isModern }) => `[name]${isModern ? '.modern' : ''}.js`,
      css: ({ isDev }) => (isDev ? '[name].css' : 'css/[name].css'),
      img: ({ isDev }) => (isDev ? '[path][name].[ext]' : 'img/[name].[ext]'),
      font: ({ isDev }) =>
        isDev ? '[path][name].[ext]' : 'fonts/[name].[ext]',
      video: ({ isDev }) =>
        isDev ? '[path][name].[ext]' : 'videos/[name].[ext]',
    },
    extractCSS: true,
    optimization: {
      splitChunks: {
        chunks: 'async',
      },
    },
    splitChunks: {
      pages: false,
      vendor: false,
      commons: false,
      runtime: false,
      layouts: false,
      cacheGroups: {
        styles: {
          name: 'styles',
          test: /\.(css|vue)$/,
          chunks: 'all',
          enforce: true,
        },
      },
    },
  },
}

const isProd = process.env.NODE_ENV === 'production'
if (!isProd) {
  config.proxy = {
    '/api/': 'http://localhost:7000',
    '/login': 'http://localhost:7000',
    '/logout': 'http://localhost:7000',
  }
}

export default config
