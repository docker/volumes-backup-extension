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

export default function TransferDialog({ ...props }) {
  console.log("CloneDialog component rendered.");
  const ddClient = useDockerDesktopClient();
  const context = useContext(MyContext);

  const [volumeName, setVolumeName] = React.useState<string>(`rpi-vol-2`);
  const [destHost, setDestHost] = React.useState<string>("192.168.1.50");
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const transferVolume = async () => {
    setActionInProgress(true);

    try {
      // TODO: type a new volume name  or list existing destination volumes using https://mui.com/material-ui/react-autocomplete/:
      // DOCKER_HOST=ssh://pi@192.168.1.50 docker volume ls --format="{{ .Name }}"

      console.log(
        `Transferring data from source volume ${context.store.volumeName} to destination volume ${volumeName} in host ${destHost}...`
      );

      // docker run --rm \
      //      -v dockprom_prometheus_data:/from alpine ash -c \
      //      "cd /from ; tar -czf - . " | \
      //      ssh 192.168.1.50 \
      //      "docker run --rm -i -v \"rpi-vol-2\":/to alpine ash -c 'cd /to ; tar -xpvzf - '"

      const transferredOutput = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${context.store.volumeName}:/from`,
        "alpine",
        "ash",
        "-c",
        `"cd /from ; tar -czf - . \" | ssh ${destHost} \"docker run --rm -i -v \"${volumeName}\":/to alpine ash -c 'cd /to ; tar -xpvzf - '"`,
      ]);
      if (transferredOutput.stderr !== "") {
        ddClient.desktopUI.toast.error(transferredOutput.stderr);
        return;
      }

      console.log(transferredOutput);

      ddClient.desktopUI.toast.success(
        `Volume ${context.store.volumeName} transferred to destination volume ${volumeName} in host ${destHost}`
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
      <DialogTitle>Transfer a volume between Docker hosts</DialogTitle>
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
          Transfers a volume. SSH must be enabled and configured between the
          source and destination Docker hosts.
        </DialogContentText>

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
              defaultValue={`rpi-vol-2`}
              spellCheck={false}
              onChange={(e) => {
                setVolumeName(e.target.value);
              }}
            />
          </Grid>
          <Grid item>
            <TextField
              required
              autoFocus
              margin="dense"
              id="dest-host"
              label="Destination host"
              fullWidth
              variant="standard"
              defaultValue={"192.168.1.50"}
              spellCheck={false}
              onChange={(e) => {
                setDestHost(e.target.value);
              }}
            />
          </Grid>
          {volumeName !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary">
                The volume will be transferred to an existing volume named{" "}
                {volumeName} in {destHost}.
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose}>Cancel</Button>
        <Button onClick={transferVolume} disabled={volumeName === ""}>
          Transfer
        </Button>
      </DialogActions>
    </Dialog>
  );
}
