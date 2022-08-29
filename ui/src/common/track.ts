import { createDockerDesktopClient } from "@docker/extension-api-client";

const analyticsEvent = "eventVolumeBackupExtension";

const ddClient = createDockerDesktopClient();

export const track = (props: any) => {
  // @ts-ignore
  ddClient.analytics?.track(analyticsEvent, props);
};
