import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import App from "./App.vue";
import "./style.css";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      component: () => import("./views/Dashboard.vue"),
    },
    {
      path: "/runbook/:id",
      component: () => import("./views/RunbookDetail.vue"),
      props: true,
    },
    {
      path: "/execute/:id",
      component: () => import("./views/ExecutionView.vue"),
      props: true,
    },
  ],
});

createApp(App).use(router).mount("#app");
