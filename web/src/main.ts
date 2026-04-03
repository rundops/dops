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
    {
      path: "/history",
      component: () => import("./views/HistoryView.vue"),
    },
    {
      path: "/history/:id",
      component: () => import("./views/HistoryDetailView.vue"),
      props: true,
    },
    {
      path: "/:pathMatch(.*)*",
      redirect: "/",
    },
  ],
});

const app = createApp(App);

app.config.errorHandler = (err, _instance, info) => {
  console.error(`[dops] Unhandled error in ${info}:`, err);
};

app.use(router).mount("#app");
