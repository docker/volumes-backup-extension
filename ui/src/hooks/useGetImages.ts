import { useEffect, useState } from "react";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import { useNotificationContext } from "../NotificationContext";
const ddClient = createDockerDesktopClient();

interface Image {
  Containers: -1;
  Created: number;
  Id: string;
  Labels: Record<string, string>;
  ParentId: string;
  RepoDigests: any;
  RepoTags: string[];
  SharedSize: number;
  Size: number;
  VirtualSize: number;
}

export const useGetImages = () => {
  const [isLoading, setIsLoading] = useState(false);
  const [data, setData] = useState<Image[]>();
  const { sendNotification } = useNotificationContext();

  useEffect(() => {
    getImages().then(setData);
  }, []);

  const getImages = async () => {
    setIsLoading(true);
    return ddClient.docker
      .listImages()
      .then((images) => {
        setIsLoading(false);
        return images;
      })
      .catch((error) => {
        sendNotification.error(
          `Failed to get images: ${error.stderr} Exit code: ${error.code}`
        );
      });
  };

  return {
    data,
    isLoading,
    getImages,
  };
};
