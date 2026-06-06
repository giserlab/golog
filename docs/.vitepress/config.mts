import { defineConfig } from 'vitepress';

export default defineConfig({
  title: 'Golog',
  description: 'Golog 一款简单、轻量的博客程序',
  head: [['link', { rel: 'icon', href: '/favicon.ico' }]],
  appearance: {
    initialValue: 'light',
  },
  locales: {
    root: {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/',
      title: 'Golog | 轻量级博客程序',
      description: 'Golog 一款简单、轻量的博客程序',
      themeConfig: {
        nav: [
          { text: '首页', link: '/' },
          { text: '指南', link: '/guide/what-is-golog' },
          { text: '下载', link: '/download' },
          { text: '博客', link: 'https://golog-demo.giserlab.cn/admin' },
        ],
        sidebar: {
          '/guide/': [
            {
              text: '介绍',
              items: [
                { text: '什么是 Golog', link: '/guide/what-is-golog' },
                { text: '功能特性', link: '/guide/features' },
                { text: '快速开始', link: '/guide/getting-started' },
              ],
            },
          ],
        },
        socialLinks: [{ icon: 'github', link: 'https://github.com/giserlab/golog' }],
        footer: {
          message: '基于 MIT 许可发布',
          copyright: 'Copyright © Golog',
        },
        search: {
          provider: 'local',
          options: {
            translations: {
              button: {
                buttonText: '搜索',
                buttonAriaLabel: '搜索',
              },
              modal: {
                displayDetails: '显示详情',
                noResultsText: '未找到结果',
                resetButtonTitle: '清除查询',
                footer: {
                  selectText: '选择',
                  navigateText: '切换',
                  closeText: '关闭',
                },
              },
            },
          },
        },
        outline: { label: '页面导航' },
        docFooter: { prev: '上一篇', next: '下一篇' },
        lastUpdated: { text: '最后更新' },
        returnToTopLabel: '回到顶部',
        sidebarMenuLabel: '菜单',
        darkModeSwitchLabel: '切换主题',
        langMenuLabel: '切换语言',
      },
    },
    en: {
      label: 'English',
      lang: 'en-US',
      link: '/en/',
      title: 'Golog | Lightweight Blog',
      description: 'Golog - A simple, lightweight blog program written in Go',
      themeConfig: {
        nav: [
          { text: 'Home', link: '/en/' },
          { text: 'Guide', link: '/en/guide/what-is-golog' },
          { text: 'Download', link: '/en/download' },
          { text: 'Blog', link: 'https://golog-demo.giserlab.cn/admin' },
        ],
        sidebar: {
          '/en/guide/': [
            {
              text: 'Introduction',
              items: [
                { text: 'What is Golog', link: '/en/guide/what-is-golog' },
                { text: 'Features', link: '/en/guide/features' },
                { text: 'Getting Started', link: '/en/guide/getting-started' },
              ],
            },
          ],
        },
        socialLinks: [{ icon: 'github', link: 'https://github.com/giserlab/golog' }],
        footer: {
          message: 'Released under the MIT License',
          copyright: 'Copyright © Golog',
        },
        search: {
          provider: 'local',
          options: {
            translations: {
              button: {
                buttonText: 'Search',
                buttonAriaLabel: 'Search',
              },
              modal: {
                displayDetails: 'Show details',
                noResultsText: 'No results found',
                resetButtonTitle: 'Clear query',
                footer: {
                  selectText: 'Select',
                  navigateText: 'Navigate',
                  closeText: 'Close',
                },
              },
            },
          },
        },
        outline: { label: 'On this page' },
        docFooter: { prev: 'Previous', next: 'Next' },
        lastUpdated: { text: 'Last updated' },
        returnToTopLabel: 'Return to top',
        sidebarMenuLabel: 'Menu',
        darkModeSwitchLabel: 'Switch theme',
        langMenuLabel: 'Change language',
      },
    },
    'zh-tw': {
      label: '繁體中文',
      lang: 'zh-TW',
      link: '/zh-tw/',
      title: 'Golog | 輕量的部落格程式',
      description: 'Golog 一款簡單、輕量的部落格程式',
      themeConfig: {
        nav: [
          { text: '首頁', link: '/zh-tw/' },
          { text: '指南', link: '/zh-tw/guide/what-is-golog' },
          { text: '下載', link: '/zh-tw/download' },
          { text: '部落格', link: 'https://golog-demo.giserlab.cn/admin' },
        ],
        sidebar: {
          '/zh-tw/guide/': [
            {
              text: '介紹',
              items: [
                { text: '什麼是 Golog', link: '/zh-tw/guide/what-is-golog' },
                { text: '功能特性', link: '/zh-tw/guide/features' },
                { text: '快速開始', link: '/zh-tw/guide/getting-started' },
              ],
            },
          ],
        },
        socialLinks: [{ icon: 'github', link: 'https://github.com/giserlab/golog' }],
        footer: {
          message: '基於 MIT 許可發布',
          copyright: 'Copyright © Golog',
        },
        search: {
          provider: 'local',
          options: {
            translations: {
              button: {
                buttonText: '搜尋',
                buttonAriaLabel: '搜尋',
              },
              modal: {
                displayDetails: '顯示詳情',
                noResultsText: '未找到結果',
                resetButtonTitle: '清除查詢',
                footer: {
                  selectText: '選擇',
                  navigateText: '切換',
                  closeText: '關閉',
                },
              },
            },
          },
        },
        outline: { label: '頁面導航' },
        docFooter: { prev: '上一篇', next: '下一篇' },
        lastUpdated: { text: '最後更新' },
        returnToTopLabel: '回到頂部',
        sidebarMenuLabel: '選單',
        darkModeSwitchLabel: '切換主題',
        langMenuLabel: '切換語言',
      },
    },
  },
});
