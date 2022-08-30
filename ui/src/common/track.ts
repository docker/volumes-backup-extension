import { createDockerDesktopClient } from "@docker/extension-api-client";

const analyticsEvent = "eventVolumeBackupExtension";

const ddClient = createDockerDesktopClient();

export const track = (props: unknown) => {
  // @ts-expect-error not there?
  ddClient.analytics?.track(analyticsEvent, props);
};
