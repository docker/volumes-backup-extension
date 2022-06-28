import React, { useEffect, useContext } from "react";
import {
  Button,
  TextField,
  Typography,
  Grid,
  Backdrop,
  CircularProgress,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";
import { isError } from "../common/isError";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function SaveDialog({ ...props }) {
  console.log("SaveDialog component rendered.");
  const ddClient = useDockerDesktopClient();

  const context = useContext(MyContext);
  const defaultImageName = `vackup-${context.store.volumeName}:latest`;
  const [imageName, setImageName] = React.useState<string>(defaultImageName);
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const saveVolume = async () => {
    setActionInProgress(true);

    const containerName = "save-volume";

    try {
      const cpOutput = await ddClient.docker.cli.exec("run", [
        `--name=${containerName}`,
        `-v=${context.store.volumeName}:/mount-volume `,
        "busybox",
        "/bin/sh",
        "-c",
        '"cp -Rp /mount-volume/. /volume-data/;"',
      ]);
      if (isError(cpOutput.stderr)) {
        ddClient.desktopUI.toast.error(cpOutput.stderr);
        return;
      }

      const psOutput = await ddClient.docker.cli.exec("ps", [
        "-aq",
        `--filter="name=${containerName}"`,
      ]);
      if (psOutput.stderr !== "") {
        ddClient.desktopUI.toast.error(psOutput.stderr);
        return;
      }

      const containerId = psOutput.lines()[0];

      const commitOutput = await ddClient.docker.cli.exec("commit", [
        containerId,
        imageName,
      ]);

      if (commitOutput.stderr !== "") {
        ddClient.desktopUI.toast.error(commitOutput.stderr);
        return;
      }

      const containerRmOutput = await ddClient.docker.cli.exec("container", [
        "rm",
        containerId,
      ]);

      if (containerRmOutput.stderr !== "") {
        ddClient.desktopUI.toast.error(containerRmOutput.stderr);
        return;
      }

      ddClient.desktopUI.toast.success(
        `Volume ${context.store.volumeName} copied into image ${imageName}, under /volume-data`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to copy volume ${context.store.volumeName} into image ${imageName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose();
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Copy the volume contents to a local image</DialogTitle>
      <DialogContent>
        <Backdrop
          sx={{
            backgroundColor: "rgba(245,244,244,0.4)",
            zIndex: (theme) => theme.zIndex.drawer + 1,
          }}
          open={actionInProgress}
        >
          <CircularProgress color="info" />
        </Backdrop>
        <DialogContentText>
          Copies the volume contents to a busybox image in the /volume-data
          directory.
        </DialogContentText>

        <Grid container direction="column" spacing={2}>
          <Grid item>
            <TextField
              required
              autoFocus
              margin="dense"
              id="image-name"
              label="Image name"
              fullWidth
              variant="standard"
              placeholder={defaultImageName}
              defaultValue={defaultImageName}
              spellCheck={false}
              onChange={(e) => {
                setImageName(e.target.value);
              }}
            />
          </Grid>

          {imageName !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
                The volume contents will be saved into the /volume-content
                directory of the image {imageName}.
              </Typography>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
                Once the operation is completed, you could see the data from a
                terminal:
              </Typography>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
                $ docker run --rm {imageName} ls /volume-data
              </Typography>
              <Typography variant="body1" color="text.secondary">
                ⚠️ This will replace any existing data inside the
                /volume-content directory of the image.
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button
          onClick={() => {
            props.onClose();
          }}
        >
          Cancel
        </Button>
        <Button onClick={saveVolume} disabled={imageName === ""}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
}
