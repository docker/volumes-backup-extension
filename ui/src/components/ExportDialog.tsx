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

export default function ExportDialog({ ...props }) {
  console.log("ExportDialog component rendered.");
  const ddClient = useDockerDesktopClient();
  const context = useContext(MyContext);

  const [fileName, setFileName] = React.useState<string>(
    `${context.store.volumeName}.tar.gz`
  );
  const [path, setPath] = React.useState<string>("");
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const selectExportDirectory = () => {
    ddClient.desktopUI.dialog
      .showOpenDialog({
        properties: ["openDirectory"],
      })
      .then((result) => {
        if (result.canceled) {
          return;
        }

        setPath(result.filePaths[0]);
      });
  };

  const exportVolume = async () => {
    setActionInProgress(true);

    console.log(
      "dockerContextName before exporting volume: ",
      props.dockerContextName
    );

    let hostPath = path;
    if (props.dockerContextName !== "default") {
      console.log("rewriting host path to /tmp");
      hostPath = "/tmp";
    }

    try {
      const args = [
        props.dockerContextName,
        "run",
        "--name=export-volume-ctr",
        "-d", // run it in the background
        `-v=${context.store.volumeName}:/vackup-volume `,
        `-v=${hostPath}:/vackup`,
        "busybox",
        "/bin/sh",
        "-c",
        `"tar -zcvf /vackup/${fileName} /vackup-volume && sleep 120"`,
      ];
      console.log(args.join(" "));
      const output = await ddClient.docker.cli.exec("--context", args);
      console.log(output);
      if (output.stderr !== "") {
        //"tar: removing leading '/' from member names\n"
        if (!output.stderr.includes("tar: removing leading")) {
          // this is an error we may want to display
          ddClient.desktopUI.toast.error(output.stderr);
          return;
        }
      }

      if (props.dockerContextName !== "default") {
        // we need to copy the backup file from the remote Docker host to our local filesystem

        hostPath = path; // restore our host path to bind mount the local filesystem

        const args = [
          props.dockerContextName, // use the docker context where the export-volume-ctr is running
          "cp",
          `export-volume-ctr:/vackup/${fileName}`,
          `${hostPath}`, // this is finally the path in our local filesystem
        ];
        console.log(args.join(" "));
        const output = await ddClient.docker.cli.exec("--context", args);
        console.log(output);
      }

      ddClient.desktopUI.toast.success(
        `Volume ${context.store.volumeName} exported to ${path}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to backup volume ${context.store.volumeName} to ${path}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      const args = [props.dockerContextName, "rm", "-f", "export-volume-ctr"];
      console.log(args.join(" "));
      const output = await ddClient.docker.cli.exec("--context", args);
      console.log(output);

      setActionInProgress(false);
      props.onClose();
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Export volume to local directory</DialogTitle>
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
          Creates a gzip'ed tarball in the selected directory from a volume.
        </DialogContentText>

        <Grid container direction="column" spacing={2}>
          <Grid item>
            <TextField
              required
              autoFocus
              margin="dense"
              id="file-name"
              label="File name"
              fullWidth
              variant="standard"
              defaultValue={`${context.store.volumeName}.tar.gz`}
              spellCheck={false}
              onChange={(e) => {
                setFileName(e.target.value);
              }}
            />
          </Grid>
          <Grid item>
            <Button variant="contained" onClick={selectExportDirectory}>
              Select directory
            </Button>
          </Grid>

          {path !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary">
                The volume will be exported to {path}/{fileName}
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose}>Cancel</Button>
        <Button
          onClick={exportVolume}
          disabled={path === "" || fileName === ""}
        >
          Export
        </Button>
      </DialogActions>
    </Dialog>
  );
}
