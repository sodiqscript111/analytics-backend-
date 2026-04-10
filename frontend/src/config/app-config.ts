import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

export const APP_CONFIG = {
  name: "Analytics Dashboard",
  version: packageJson.version,
  copyright: `© ${currentYear}, Analytics Dashboard.`,
  meta: {
    title: "Analytics Dashboard",
    description:
      "A lightweight realtime analytics dashboard for live event monitoring and event search.",
  },
};
