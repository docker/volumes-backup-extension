import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useContext, useState } from "react";
import { MyContext } from "..";
import { useNotificationContext } from "../NotificationContext";

const ddClient = createDockerDesktopClient();

export const useExportVolume = () => {
  const [isLoading, setIsLoading] = useState(false);
  const { sendNotification } = useNotificationContext();
  const context = useContext(MyContext);
  const selectedVolumeName = context.store.volume?.volumeName;

  const exportVolume = ({
    path,
    fileName,
  }: {
    path: string;
    fileName: string;
  }) => {
    setIsLoading(true);

    return ddClient.extension.vm.service
      .get(
        `/volumes/${selectedVolumeName}/export?path=${path}&fileName=${fileName}`
      )
      .then((_: any) => {
        sendNotification.info(`Volume ${selectedVolumeName} exported to ${path}`);
      })
      .catch((error) => {
        sendNotification.error(
          `Failed to backup volume ${selectedVolumeName} to ${path}: ${error.message}. HTTP status code: ${error.statusCode}`
        );
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return {
    exportVolume,
    isLoading,
  };
};
