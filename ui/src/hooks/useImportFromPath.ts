import { useState } from "react";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useNotificationContext } from "../NotificationContext";
const ddClient = createDockerDesktopClient();

export const useImportFromPath = () => {
  const [isInProgress, setIsInProgress] = useState(false);
  const { sendNotification } = useNotificationContext();

  const importVolume = async ({
    volumeName,
    path,
  }: {
    volumeName: string;
    path: string;
  }) => {
    setIsInProgress(true);
    return ddClient.extension.vm.service
      .get(`/volumes/${volumeName}/import?path=${path}`)
      .then((_: any) => {
        setIsInProgress(false);
        sendNotification(`File ${path} imported into volume ${volumeName}`, [
          {
            name: "See volume",
            onClick: () => ddClient.desktopUI.navigate.viewVolume(volumeName),
          },
        ]);
      })
      .catch((error) => {
        setIsInProgress(false);
        sendNotification(
          `Failed to import file ${path} into volume ${volumeName}: ${error.message}. HTTP status code: ${error.statusCode}`,
          [],
          "error"
        );
      });
  };

  return {
    importVolume,
    isInProgress,
  };
};
