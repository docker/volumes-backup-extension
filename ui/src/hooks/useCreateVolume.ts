import { useState } from "react";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useNotificationContext } from "../NotificationContext";
const ddClient = createDockerDesktopClient();

export const useCreateVolume = () => {
  const { sendNotification } = useNotificationContext();
  const [isInProgress, setIsInProgress] = useState(false);

  const createVolume = async (volumeName: string) => {
    setIsInProgress(true);
    return ddClient.docker.cli
      .exec("volume", ["create", volumeName])
      .then((createVolumeOutput) => {
        if (createVolumeOutput.stderr !== "") {
          sendNotification(createVolumeOutput.stderr);
        }
        setIsInProgress(false);
        return createVolumeOutput.lines();
      })
      .catch((error) => {
        sendNotification(
          `Failed to create volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
        );
      });
  };

  return {
    createVolume,
    isInProgress,
  };
};
