import React, { useContext } from "react";
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

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function CloneDialog({ ...props }) {
  const ddClient = useDockerDesktopClient();
  const context = useContext(MyContext);

  const [volumeName, setVolumeName] = React.useState<string>(
    `${context.store.volumeName}-cloned`
  );
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const cloneVolume = async () => {
    setActionInProgress(true);

    try {
      // TODO: check if destination volume already exists
      const createVolumeOutput = await ddClient.docker.cli.exec("volume", [
        "create",
        volumeName,
      ]);
      if (createVolumeOutput.stderr !== "") {
        ddClient.desktopUI.toast.error(createVolumeOutput.stderr);
        return;
      }

      const cloneOutput = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${context.store.volumeName}:/from`,
        `-v=${volumeName}:/to`,
        "alpine",
        "ash",
        "-c",
        '"cd /from ; cp -av . /to"',
      ]);
      if (cloneOutput.stderr !== "") {
        ddClient.desktopUI.toast.error(cloneOutput.stderr);
        return;
      }

      ddClient.desktopUI.toast.success(
        `Volume ${context.store.volumeName} cloned to destination volume ${volumeName}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to clone volume ${context.store.volumeName} to destinaton volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose();
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Clone a volume</DialogTitle>
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
        <DialogContentText>Clones a volume.</DialogContentText>

        <Grid container direction="column" spacing={2}>
          <Grid item>
            <TextField
              required
              autoFocus
              margin="dense"
              id="volume-name"
              label="Volume name"
              fullWidth
              variant="standard"
              defaultValue={`${context.store.volumeName}-cloned`}
              spellCheck={false}
              onChange={(e) => {
                setVolumeName(e.target.value);
              }}
            />
          </Grid>

          {volumeName !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary">
                The volume will be cloned to a new volume named {volumeName}.
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose}>Cancel</Button>
        <Button onClick={cloneVolume} disabled={volumeName === ""}>
          Clone
        </Button>
      </DialogActions>
    </Dialog>
  );
}
