import { defineConfig } from "vitepress";

export default defineConfig({
  title: "Golog | 轻量的博客程序",
  description: "Golog 一款简单、轻量的博客程序",
  lang: "zh-CN",
  head: [["link", { rel: "icon", href: "/favicon.ico" }]],
  themeConfig: {
    nav: [
      { text: "首页", link: "/" },
      { text: "指南", link: "/guide/what-is-golog" },
      { text: "下载", link: "/download" },
      { text: "在线示例", link: "https://golog.giserlab.cn" },
    ],
    sidebar: {
      "/guide/": [
        {
          text: "介绍",
          items: [
            { text: "什么是 Golog", link: "/guide/what-is-golog" },
            { text: "功能特性", link: "/guide/features" },
            { text: "快速开始", link: "/guide/getting-started" },
          ],
        },
      ],
    },
    socialLinks: [{ icon: "github", link: "https://github.com/WShihan/golog" }],
    footer: {
      message: "基于 MIT 许可发布",
      copyright: "Copyright © Golog 贡献者",
    },
    search: {
      provider: "local",
      options: {
        locales: {
          root: {
            translations: {
              button: {
                buttonText: "搜索",
                buttonAriaLabel: "搜索",
              },
              modal: {
                displayDetails: "显示详情",
                noResultsText: "未找到结果",
                resetButtonTitle: "清除查询",
                footer: {
                  selectText: "选择",
                  navigateText: "切换",
                  closeText: "关闭",
                },
              },
            },
          },
        },
      },
    },
    outline: {
      label: "页面导航",
    },
    docFooter: {
      prev: "上一篇",
      next: "下一篇",
    },
    lastUpdated: {
      text: "最后更新",
    },
    returnToTopLabel: "回到顶部",
    sidebarMenuLabel: "菜单",
    darkModeSwitchLabel: "切换主题",
    langMenuLabel: "切换语言",
  },
});
