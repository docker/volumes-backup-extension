import { useState } from "react";
import { createDockerDesktopClient } from "@docker/extension-api-client";
const ddClient = createDockerDesktopClient();

export const useCreateVolume = () => {
  const [isInProgress, setIsInProgress] = useState(false);

  const createVolume = async (volumeName: string) => {
    setIsInProgress(true);
    return ddClient.docker.cli
      .exec("volume", ["create", volumeName])
      .then((createVolumeOutput) => {
        if (createVolumeOutput.stderr !== "") {
          ddClient.desktopUI.toast.error(createVolumeOutput.stderr);
        }
        setIsInProgress(false);
        return createVolumeOutput.lines();
      })
      .catch((error) => {
        ddClient.desktopUI.toast.error(
          `Failed to create volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
        );
      });
  };

  return {
    createVolume,
    isInProgress,
  };
};
