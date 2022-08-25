import {createDockerDesktopClient} from "@docker/extension-api-client";
import {useContext, useState} from "react";
import {MyContext} from "..";
import {useNotificationContext} from "../NotificationContext";

const ddClient = createDockerDesktopClient();

interface Props {
    onFinish(): void;
}

export const usePushVolumeToRegistry = ({onFinish}: Props) => {
    const [isLoading, setIsLoading] = useState(false);
    const context = useContext(MyContext);
    const {sendNotification} = useNotificationContext();

    const pushVolumeToRegistry = ({imageName}: { imageName: string }) => {
        setIsLoading(true);

        ddClient.extension.host.cli
            .exec("docker-credentials-client", ["get-creds", imageName])
            .then((result) => {
                let data = {reference: imageName, base64EncodedAuth: ""};

                const base64EncodedAuth = result.stdout;
                // If the decoded base64 string is "e30=", it means is an empty JSON "{}"
                if (base64EncodedAuth !== "e30=") {
                    data.base64EncodedAuth = base64EncodedAuth;
                }

                const requestConfig = {
                    method: "POST",
                    url: `/volumes/${context.store.volume.volumeName}/push`,
                    headers: {},
                    data: data,
                };

                ddClient.extension.vm.service
                    .request(requestConfig)
                    .then((result) => {
                        sendNotification.info(
                            `Volume ${context.store.volume.volumeName} pushed as ${imageName} to registry`
                        );
                    })
                    .catch((error) => {
                        console.error(error);
                        if (
                            error?.message.includes(
                                "denied: requested access to the resource is denied"
                            )
                        ) {
                            sendNotification.error(
                                `Access denied when trying to push to ${imageName}.
                          Are you logged in? If so, check your permissions.`
                            );
                        } else {
                            sendNotification.error(
                                `Failed to push volume ${context.store.volume.volumeName} as ${imageName} to registry: ${error.message}. HTTP status code: ${error.statusCode}`
                            );
                        }
                    }).finally(() => {
                    console.log("************ onFinish !!!!!!!!!!")
                    onFinish();
                });
            })
            .catch((error) => {
                console.error(error);
                sendNotification.error(
                    `Failed to get Docker credentials when pushing volume ${context.store.volume.volumeName} as ${imageName} to registry: ${error.message}. HTTP status code: ${error.statusCode}`
                );
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
