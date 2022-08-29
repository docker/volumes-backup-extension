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
      .exec("docker-credentials-client", ["get-creds", imageName])
      .then((result) => {
        let data = { reference: imageName, base64EncodedAuth: "" };

        const base64EncodedAuth = result.stdout;
        // If the decoded base64 string is "e30=", it means is an empty JSON "{}"
        if (base64EncodedAuth !== "e30=") {
          data.base64EncodedAuth = base64EncodedAuth;
        }

        const requestConfig = {
          method: "POST",
          url: `/volumes/${volumeId || context.store.volume.volumeName}/pull`,
          headers: {},
          data: data,
        };

        ddClient.extension.vm.service
          .request(requestConfig)
          .then((result) => {
            sendNotification.info(
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
            sendNotification.error(
              `Failed to pull volume ${
                volumeId || context.store.volume.volumeName
              } as ${imageName} from registry: ${
                error.message
              }. HTTP status code: ${error.statusCode}`
            );
          });
      })
      .catch((error) => {
        console.error(error);
        sendNotification.error(
          `Failed to get Docker credentials when pulling volume ${
            volumeId || context.store.volume.volumeName
          } as ${imageName} from registry: ${
            error.message
          }. HTTP status code: ${error.statusCode}`
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
