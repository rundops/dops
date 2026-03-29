import DefaultTheme from "vitepress/theme";
import "./custom.css";
import HeroAnimation from "./HeroAnimation.vue";
import { h } from "vue";

export default {
  extends: DefaultTheme,
  Layout() {
    return h(DefaultTheme.Layout, null, {
      "home-hero-image": () => h(HeroAnimation),
    });
  },
};
