import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useContext, useState } from "react";
import { MyContext } from "..";
import { useNotificationContext } from "../NotificationContext";

const ddClient = createDockerDesktopClient();

export const useExportToImage = () => {
  const [isLoading, setIsLoading] = useState(false);
  const { sendNotification } = useNotificationContext();
  const context = useContext(MyContext);
  const selectedVolumeName = context.store.volume?.volumeName;

  const exportToImage = ({ imageName }: { imageName: string }) => {
    setIsLoading(true);

    return ddClient.extension.vm.service
      .get(
        `/volumes/${context.store.volume.volumeName}/save?image=${imageName}`
      )
      .then((_: any) => {
        sendNotification(
          `Volume ${selectedVolumeName} exported to ${imageName}`,
          {
            name: "See image",
            onClick: async () => {
              const [_, tag] = imageName.split(":");
              const image = (await ddClient.docker.cli.exec('image', ['inspect', imageName])).parseJsonObject();
              ddClient.desktopUI.navigate.viewImage(image[0].Id, tag || "latest");
            },
          }
        );
      })
      .catch((error) => {
        sendNotification(
          `Failed to backup volume ${selectedVolumeName} to ${imageName}: ${error.message}. HTTP status code: ${error.statusCode}`
        );
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return {
    exportToImage,
    isLoading,
  };
};
