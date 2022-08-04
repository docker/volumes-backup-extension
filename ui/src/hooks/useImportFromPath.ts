import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useState } from "react";
const ddClient = createDockerDesktopClient();

export const useImportFromPath = () => {
  const [isInProgress, setIsInProgress] = useState(false);

  const importVolume = async ({ volumeName, path }: { volumeName: string, path: string }) => {
    setIsInProgress(true);
    return ddClient.extension.vm.service
      .get(`/volumes/${volumeName}/import?path=${path}`)
      .then((_: any) => {
        setIsInProgress(false);
        ddClient.desktopUI.toast.success(
          `File ${path} imported into volume ${volumeName}`
        );
      })
      .catch((error) => {
        setIsInProgress(false);
        ddClient.desktopUI.toast.error(
          `Failed to import file ${path} into volume ${volumeName}: ${error.message}. HTTP status code: ${error.statusCode}`
        );
      });
  };

  return {
    importVolume, isInProgress
  }
};
