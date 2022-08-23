import { useContext, useState } from "react";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import { MyContext } from "..";
import { useNotificationContext } from "../NotificationContext";

const ddClient = createDockerDesktopClient();

export const usePullFromRegistry = () => {
  const [isLoading, setIsLoading] = useState(false);
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();

  const pullFromRegistry = ({
    imageName,
    volumeId,
  }: {
    imageName: string;
    volumeId: string;
  }) => {
    setIsLoading(true);

    return ddClient.extension.host.cli
      .exec("volumes-share-client", [
        "--extension-dir",
        process.env["REACT_APP_EXTENSION_INSTALLATION_DIR_NAME"],
        "pull",
        imageName,
        volumeId || context.store.volume.volumeName,
      ])
      .then((result) => {
        sendNotification(
          `Volume ${
            volumeId || context.store.volume.volumeName
          } pulled as ${imageName} from registry`,
          [
            {
              name: "See volume",
              onClick: () =>
                ddClient.desktopUI.navigate.viewVolume(
                  volumeId || context.store.volume.volumeName
                ),
            },
          ]
        );
      })
      .catch((error) => {
        console.error(error);
        sendNotification(
          `Failed to pull volume ${
            volumeId || context.store.volume.volumeName
          } as ${imageName} from registry: ${
            error.message
          }. HTTP status code: ${error.statusCode}`,
          [],
          "error"
        );
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return {
    pullFromRegistry,
    isLoading,
  };
};
