import { defineConfig } from "vitepress";

export default defineConfig({
  title: "dops",
  description: "a runbook toolkit for operators and AI agents",
  base: "/dops/",

  head: [
    ["link", { rel: "icon", type: "image/png", href: "/dops/logo.png" }],
  ],

  appearance: "dark",

  themeConfig: {
    logo: "/logo.png",
    siteTitle: "dops",

    nav: [
      { text: "Guides", link: "/guides/getting-started" },
      {
        text: "Reference",
        items: [
          { text: "CLI Commands", link: "/reference/cli" },
          { text: "Configuration", link: "/reference/configuration" },
          { text: "Keyboard Shortcuts", link: "/reference/keyboard-shortcuts" },
        ],
      },
      {
        text: "GitHub",
        link: "https://github.com/rundops/dops",
      },
    ],

    sidebar: {
      "/guides/": [
        {
          text: "Guides",
          items: [
            { text: "Getting Started", link: "/guides/getting-started" },
            { text: "Web UI", link: "/guides/web-ui" },
            { text: "Creating Runbooks", link: "/guides/runbooks" },
            { text: "MCP / AI Agents", link: "/guides/mcp" },
            { text: "Verification", link: "/guides/verification" },
          ],
        },
      ],
      "/reference/": [
        {
          text: "CLI Commands",
          link: "/reference/cli",
          items: [
            { text: "dops", link: "/reference/cli/dops" },
            { text: "dops init", link: "/reference/cli/dops-init" },
            { text: "dops run", link: "/reference/cli/dops-run" },
            { text: "dops open", link: "/reference/cli/dops-open" },
            { text: "dops config", link: "/reference/cli/dops-config" },
            { text: "dops catalog", link: "/reference/cli/dops-catalog" },
            { text: "dops mcp", link: "/reference/cli/dops-mcp" },
            { text: "dops completion", link: "/reference/cli/dops-completion" },
            { text: "dops version", link: "/reference/cli/dops-version" },
          ],
        },
        {
          text: "Reference",
          items: [
            { text: "Configuration", link: "/reference/configuration" },
            { text: "Keyboard Shortcuts", link: "/reference/keyboard-shortcuts" },
          ],
        },
      ],
    },

    socialLinks: [
      {
        icon: "github",
        link: "https://github.com/rundops/dops",
      },
    ],

    footer: {
      message:
        'Released under the <a href="https://github.com/rundops/dops/blob/main/LICENSE">MIT License</a>.',
      copyright:
        'Copyright © 2025 Mason Huemmer',
    },

    editLink: {
      pattern:
        "https://github.com/rundops/dops/edit/main/docs/:path",
      text: "Edit this page on GitHub",
    },

    outline: { level: "deep", label: "On this page" },

    search: {
      provider: "local",
    },
  },
});
