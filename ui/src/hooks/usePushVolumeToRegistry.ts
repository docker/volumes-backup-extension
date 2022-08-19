import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useContext, useState } from "react";
import { MyContext } from "..";
import { useNotificationContext } from "../NotificationContext";

const ddClient = createDockerDesktopClient();

export const usePushVolumeToRegistry = () => {
  const [isLoading, setIsLoading] = useState(false);
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();

  const pushVolumeToRegistry = ({ imageName }: { imageName: string }) => {
    setIsLoading(true);

    return ddClient.extension.host.cli
      .exec("volumes-share-client", [
        "--extension-dir",
        process.env["REACT_APP_EXTENSION_INSTALLATION_DIR_NAME"],
        "push",
        imageName,
        context.store.volume.volumeName,
      ])
      .then((result) => {
        sendNotification(
          `Volume ${context.store.volume.volumeName} pushed as ${imageName} to registry`
        );
      })
      .catch((error) => {
        if (error?.stderr.includes('denied: requested access to the resource is denied')) {
          sendNotification(
            `Access denied when trying to push to ${imageName}.
            Are you logged in? If so, check your permissions.`
          )
        } else {
          sendNotification(
            `Failed to push volume ${context.store.volume.volumeName} as ${imageName} to registry: ${error.message}. HTTP status code: ${error.statusCode}`
          );
        }
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return {
    pushVolumeToRegistry,
    isLoading,
  };
};
