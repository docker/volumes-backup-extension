import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useState } from "react";
import { useNotificationContext } from "../NotificationContext";
const ddClient = createDockerDesktopClient();

interface ILoadImage {
  volumeName: string;
  imageName: string;
}
export const useImportFromImage = () => {
  const [isInProgress, setIsInProgress] = useState(false);
  const { sendNotification } = useNotificationContext();

  const loadImage = async ({ volumeName, imageName }: ILoadImage) => {
    setIsInProgress(true);

    return ddClient.extension.vm.service
      .get(`/volumes/${volumeName}/load?image=${imageName}`)
      .then((_: any) => {
        setIsInProgress(false);
        sendNotification(
          `Copied /volume-data from image ${imageName} into volume ${volumeName}`,
          [
            {
              name: "See volume",
              onClick: () => ddClient.desktopUI.navigate.viewVolume(volumeName),
            },
          ]
        );
      })
      .catch((error) => {
        setIsInProgress(false);
        sendNotification(
          `Failed to copy /volume-data from image ${imageName} to into volume ${volumeName}: ${error.message}. HTTP status code: ${error.statusCode}`
        );
      });
  };

  return {
    loadImage,
    isInProgress,
  };
};
