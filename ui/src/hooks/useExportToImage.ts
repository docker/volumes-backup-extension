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

  const navigateToImage = (imageName: string, imageId: string, tag: string) => {
    ddClient.desktopUI.navigate
      .viewImage(imageId, tag || "latest")
      .catch(() => {
        sendNotification(`Couldn't navigate to image ${imageName}`, [
          {
            name: "Try again",
            onClick: () => navigateToImage(imageName, imageId, tag),
          },
          {
            name: "Dismiss",
          },
        ]);
      });
  };

  const exportToImage = ({ imageName }: { imageName: string }) => {
    setIsLoading(true);

    return ddClient.extension.vm.service
      .get(
        `/volumes/${context.store.volume.volumeName}/save?image=${imageName}`
      )
      .then((_: any) => {
        sendNotification(
          `Volume ${selectedVolumeName} exported to ${imageName}`,
          [
            {
              name: "See image",
              onClick: async () => {
                const [_, tag] = imageName.split(":");
                const image = (
                  await ddClient.docker.cli.exec("image", [
                    "inspect",
                    imageName,
                  ])
                ).parseJsonObject();
                navigateToImage(imageName, image[0].Id, tag);
              },
            },
          ]
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
