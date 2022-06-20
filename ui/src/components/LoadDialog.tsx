import React, { useEffect } from "react";
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

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function LoadDialog({ ...props }) {
  const [imageName, setImageName] = React.useState<string>(props.volumeName);
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const ddClient = useDockerDesktopClient();

  useEffect(() => {
    setImageName(`vackup-${props.volumeName}:latest`);
  }, [props.volumeName]);

  const loadImage = async () => {
    setActionInProgress(true);

    try {
      const output = await ddClient.docker.cli.exec("run", [
        `--rm`,
        `-v=${props.volumeName}:/mount-volume `,
        imageName,
        "/bin/sh",
        "-c",
        '"cp -Rp /volume-data/. /mount-volume/;"',
      ]);
      if (output.stderr !== "") {
        ddClient.desktopUI.toast.error(output.stderr);
        return;
      }

      ddClient.desktopUI.toast.success(
        `Copied /volume-data from image ${imageName} into volume ${props.volumeName}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to copy /volume-data from image ${imageName} to into volume ${props.volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose();
      setImageName("");
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Load a directory from an image to a volume</DialogTitle>
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
          Copies /volume-data contents from an image to a volume
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
              placeholder={`my-image:latest`}
              spellCheck={false}
              onChange={(e) => {
                setImageName(e.target.value);
              }}
            />
          </Grid>

          {imageName !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
                The /volume-data from image {imageName} will be copied to volume{" "}
                {props.volumeName}.
              </Typography>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
                Once the operation is completed, you can inspect the volume
                contents of {props.volumeName} or export its content into a
                local directory.
              </Typography>
              <Typography variant="body1" color="text.secondary">
                ⚠️ This will replace any existing data inside the volume.
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button
          onClick={() => {
            props.onClose();
            setImageName("");
          }}
        >
          Cancel
        </Button>
        <Button onClick={loadImage} disabled={imageName === ""}>
          Load
        </Button>
      </DialogActions>
    </Dialog>
  );
}
