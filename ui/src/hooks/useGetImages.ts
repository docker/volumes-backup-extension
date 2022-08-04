import { useEffect, useState } from "react";
import { createDockerDesktopClient } from "@docker/extension-api-client";
const ddClient = createDockerDesktopClient();

export const useGetImages = () => {
  const [isLoading, setIsLoading] = useState(false);
  const [data, setData] = useState();

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
        ddClient.desktopUI.toast.error(
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
